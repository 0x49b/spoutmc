package io.spoutmc.velocityplayers.api;

import com.google.gson.Gson;
import com.google.gson.JsonObject;
import com.sun.net.httpserver.Headers;
import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.velocitypowered.api.proxy.Player;
import io.spoutmc.velocityplayers.model.PlayerRecord;
import io.spoutmc.velocityplayers.model.PluginConfig;
import io.spoutmc.velocityplayers.service.ChatService;
import io.spoutmc.velocityplayers.service.PlayerStateService;
import io.spoutmc.velocityplayers.util.PluginUtils;
import net.kyori.adventure.text.Component;
import org.slf4j.Logger;

import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.io.Reader;
import java.net.URI;
import java.net.URLDecoder;
import java.nio.charset.StandardCharsets;
import java.util.Objects;
import java.util.Optional;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;

public final class BridgeApiHandler implements HttpHandler {
    private final Logger logger;
    private final Gson gson;
    private final PluginConfig config;
    private final PlayerStateService playerStateService;
    private final ChatService chatService;
    private final Set<SseClient> sseClients = ConcurrentHashMap.newKeySet();

    public BridgeApiHandler(
            Logger logger,
            Gson gson,
            PluginConfig config,
            PlayerStateService playerStateService,
            ChatService chatService
    ) {
        this.logger = logger;
        this.gson = gson;
        this.config = config;
        this.playerStateService = playerStateService;
        this.chatService = chatService;
    }

    @Override
    public void handle(HttpExchange exchange) throws IOException {
        try {
            String method = exchange.getRequestMethod();
            URI uri = exchange.getRequestURI();
            String path = uri.getPath();

            if ("OPTIONS".equalsIgnoreCase(method)) {
                addCors(exchange.getResponseHeaders());
                exchange.sendResponseHeaders(204, -1);
                return;
            }

            if ("/healthz".equals(path)) {
                sendJson(exchange, 200, "{\"status\":\"ok\"}");
                return;
            }

            if (!isAuthorized(exchange)) {
                sendJson(exchange, 401, "{\"error\":\"unauthorized\"}");
                return;
            }

            if ("GET".equalsIgnoreCase(method) && "/players".equals(path)) {
                sendJson(exchange, 200, gson.toJson(playerStateService.snapshotPlayers()));
                return;
            }

            if ("GET".equalsIgnoreCase(method) && "/players/stream".equals(path)) {
                handlePlayersStream(exchange);
                return;
            }

            if ("GET".equalsIgnoreCase(method) && path.startsWith("/players/") && path.endsWith("/chat")) {
                handlePlayerAction(exchange, path);
                return;
            }

            if ("POST".equalsIgnoreCase(method) && path.startsWith("/players/")) {
                handlePlayerAction(exchange, path);
                return;
            }

            sendJson(exchange, 404, "{\"error\":\"not found\"}");
        } catch (Throwable e) {
            if (isBrokenPipe(e)) {
                logger.debug("[SpoutPlayers] Client disconnected: {}", e.getMessage());
            } else {
                logger.error("[SpoutPlayers] API handler error", e);
            }
            try {
                sendJson(exchange, 500, "{\"error\":\"internal server error\"}");
            } catch (Throwable writeError) {
                // Expected when client disconnected or headers already sent
                logger.debug("[SpoutPlayers] Could not write error response: {}", writeError.getMessage());
                try {
                    exchange.close();
                } catch (Throwable ignored) {
                }
            }
        }
    }

    public void broadcastSnapshot() {
        String payload = gson.toJson(playerStateService.snapshotPlayers());
        for (SseClient client : sseClients) {
            if (!client.sendSse(payload)) {
                sseClients.remove(client);
                client.close();
            }
        }
    }

    public void closeAllSseClients() {
        for (SseClient client : sseClients) {
            client.close();
        }
        sseClients.clear();
    }

    private void handlePlayersStream(HttpExchange exchange) throws IOException {
        Headers headers = exchange.getResponseHeaders();
        addCors(headers);
        headers.set("Content-Type", "text/event-stream");
        headers.set("Cache-Control", "no-cache");
        headers.set("Connection", "keep-alive");
        exchange.sendResponseHeaders(200, 0);

        SseClient client = new SseClient(exchange);
        sseClients.add(client);
        client.sendSse(gson.toJson(playerStateService.snapshotPlayers()));

        while (!client.isClosed()) {
            try {
                Thread.sleep(15000);
                if (!client.sendComment("ping")) {
                    break;
                }
            } catch (InterruptedException interruptedException) {
                Thread.currentThread().interrupt();
                break;
            }
        }

        sseClients.remove(client);
        client.close();
    }

    private void handlePlayerAction(HttpExchange exchange, String path) throws IOException {
        String[] segments = path.split("/");
        if (segments.length < 4) {
            sendJson(exchange, 400, "{\"error\":\"invalid path\"}");
            return;
        }

        String playerName = URLDecoder.decode(segments[2], StandardCharsets.UTF_8);
        String action = segments[3];
        JsonObject body = readJsonBody(exchange);

        if ("chat".equals(action)) {
            sendJson(exchange, 200, gson.toJson(chatService.getChat(playerName)));
            return;
        }

        if ("message".equals(action)) {
            String message = getJsonString(body, "message");
            if (message == null || message.isBlank()) {
                sendJson(exchange, 400, "{\"error\":\"message is required\"}");
                return;
            }
            String sender = getJsonString(body, "sender");
            String role = getJsonString(body, "role");

            Optional<Player> player = playerStateService.findOnlinePlayer(playerName);
            if (player.isEmpty()) {
                sendJson(exchange, 404, "{\"error\":\"player not online\"}");
                return;
            }

            chatService.sendStaffMessage(player.get(), sender, role, message);
            sendJson(exchange, 202, "{\"status\":\"message sent\"}");
            return;
        }

        if ("kick".equals(action)) {
            String reason = getJsonString(body, "reason");
            if (reason == null || reason.isBlank()) {
                reason = "Kicked by admin";
            }

            Optional<Player> player = playerStateService.findOnlinePlayer(playerName);
            if (player.isEmpty()) {
                sendJson(exchange, 404, "{\"error\":\"player not online\"}");
                return;
            }

            player.get().disconnect(Component.text(reason));
            playerStateService.markPlayerDisconnected(player.get());
            playerStateService.saveState();
            broadcastSnapshot();
            sendJson(exchange, 202, "{\"status\":\"player kicked\"}");
            return;
        }

        if ("ban".equals(action)) {
            String reason = getJsonString(body, "reason");
            if (reason == null || reason.isBlank()) {
                reason = "Banned by admin";
            }

            playerStateService.banPlayer(playerName, reason);

            Optional<Player> player = playerStateService.findOnlinePlayer(playerName);
            if (player.isPresent()) {
                player.get().disconnect(Component.text("You are banned: " + reason));
                playerStateService.markPlayerDisconnected(player.get());
            } else {
                PlayerRecord record = playerStateService.ensurePlayerRecord(playerName);
                if (record.name == null || record.name.isBlank()) {
                    record.name = playerName;
                }
                record.currentServer = "";
            }

            playerStateService.saveState();
            broadcastSnapshot();
            sendJson(exchange, 202, "{\"status\":\"player banned\"}");
            return;
        }

        if ("unban".equals(action)) {
            if (!playerStateService.unbanPlayer(playerName)) {
                sendJson(exchange, 404, "{\"error\":\"player is not banned\"}");
                return;
            }

            PlayerRecord record = playerStateService.ensurePlayerRecord(playerName);
            if (record.name == null || record.name.isBlank()) {
                record.name = playerName;
            }

            playerStateService.saveState();
            broadcastSnapshot();
            sendJson(exchange, 202, "{\"status\":\"player unbanned\"}");
            return;
        }

        sendJson(exchange, 404, "{\"error\":\"unsupported action\"}");
    }

    private JsonObject readJsonBody(HttpExchange exchange) {
        try (InputStream body = exchange.getRequestBody();
             Reader reader = new InputStreamReader(body, StandardCharsets.UTF_8)) {
            JsonObject json = gson.fromJson(reader, JsonObject.class);
            return json == null ? new JsonObject() : json;
        } catch (Exception e) {
            return new JsonObject();
        }
    }

    private String getJsonString(JsonObject json, String key) {
        if (json == null || !json.has(key) || json.get(key).isJsonNull()) {
            return null;
        }
        return json.get(key).getAsString();
    }

    private boolean isAuthorized(HttpExchange exchange) {
        if (config.token == null || config.token.isBlank()) {
            return true;
        }
        String auth = exchange.getRequestHeaders().getFirst("Authorization");
        if (auth == null) {
            return false;
        }
        return Objects.equals(auth, "Bearer " + config.token);
    }

    private static boolean isBrokenPipe(Throwable t) {
        while (t != null) {
            if (t instanceof IOException && t.getMessage() != null
                    && (t.getMessage().contains("Broken pipe") || t.getMessage().contains("Connection reset"))) {
                return true;
            }
            t = t.getCause();
        }
        return false;
    }

    private void sendJson(HttpExchange exchange, int statusCode, String payload) throws IOException {
        byte[] bytes = payload.getBytes(StandardCharsets.UTF_8);
        Headers headers = exchange.getResponseHeaders();
        addCors(headers);
        headers.set("Content-Type", "application/json; charset=utf-8");
        exchange.sendResponseHeaders(statusCode, bytes.length);
        try (OutputStream out = exchange.getResponseBody()) {
            out.write(bytes);
            out.flush();
        }
    }

    private void addCors(Headers headers) {
        headers.set("Access-Control-Allow-Origin", "*");
        headers.set("Access-Control-Allow-Headers", "Authorization, Content-Type");
        headers.set("Access-Control-Allow-Methods", "GET, POST, OPTIONS");
    }
}
