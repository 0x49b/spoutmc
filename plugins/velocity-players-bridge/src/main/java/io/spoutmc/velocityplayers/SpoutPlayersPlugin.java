package io.spoutmc.velocityplayers;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.JsonObject;
import com.google.inject.Inject;
import com.sun.net.httpserver.Headers;
import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;
import com.velocitypowered.api.event.ResultedEvent;
import com.velocitypowered.api.event.Subscribe;
import com.velocitypowered.api.event.connection.DisconnectEvent;
import com.velocitypowered.api.event.connection.LoginEvent;
import com.velocitypowered.api.event.connection.PostLoginEvent;
import com.velocitypowered.api.event.player.ServerConnectedEvent;
import com.velocitypowered.api.event.proxy.ProxyInitializeEvent;
import com.velocitypowered.api.event.proxy.ProxyShutdownEvent;
import com.velocitypowered.api.plugin.Plugin;
import com.velocitypowered.api.plugin.annotation.DataDirectory;
import com.velocitypowered.api.proxy.Player;
import com.velocitypowered.api.proxy.ProxyServer;
import net.kyori.adventure.text.Component;
import org.slf4j.Logger;

import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.io.Reader;
import java.net.InetSocketAddress;
import java.net.URI;
import java.net.URLDecoder;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Instant;
import java.time.format.DateTimeFormatter;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import java.util.Properties;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.Executors;

@Plugin(
        id = "spoutmc-players",
        name = "SpoutMC Players Bridge",
        version = "0.1.0",
        description = "Tracks players and exposes HTTP/SSE API for SpoutMC",
        authors = {"SpoutMC"}
)
public final class SpoutPlayersPlugin {
    private static final String CONFIG_FILE = "config.properties";
    private static final String STATE_FILE = "state.json";

    private final ProxyServer proxy;
    private final Logger logger;
    private final Path dataDirectory;
    private final Gson gson;

    private final Map<String, PlayerRecord> playersByName = new ConcurrentHashMap<>();
    private final Map<String, BanRecord> bansByName = new ConcurrentHashMap<>();
    private final Set<SseClient> sseClients = ConcurrentHashMap.newKeySet();

    private HttpServer httpServer;
    private PluginConfig config;

    @Inject
    public SpoutPlayersPlugin(ProxyServer proxy, Logger logger, @DataDirectory Path dataDirectory) {
        this.proxy = proxy;
        this.logger = logger;
        this.dataDirectory = dataDirectory;
        this.gson = new GsonBuilder().setPrettyPrinting().create();
    }

    @Subscribe
    public void onInitialize(ProxyInitializeEvent event) {
        try {
            Files.createDirectories(dataDirectory);
            this.config = loadConfig();
            loadState();
            seedOnlinePlayers();
            startHttpServer();
            broadcastSnapshot();
            logger.info("[SpoutPlayers] Plugin initialized on {}:{}", config.bindHost, config.port);
        } catch (Exception e) {
            logger.error("[SpoutPlayers] Failed to initialize plugin", e);
        }
    }

    @Subscribe
    public void onShutdown(ProxyShutdownEvent event) {
        stopHttpServer();
        saveState();
        logger.info("[SpoutPlayers] Plugin shutdown complete");
    }

    @Subscribe
    public void onLogin(LoginEvent event) {
        String playerName = normalizeName(event.getPlayer().getUsername());
        BanRecord ban = bansByName.get(playerName);
        if (ban != null) {
            String reason = ban.reason == null || ban.reason.isBlank() ? "Banned from this network" : ban.reason;
            event.setResult(ResultedEvent.ComponentResult.denied(Component.text("You are banned: " + reason)));
        }
    }

    @Subscribe
    public void onPostLogin(PostLoginEvent event) {
        Player player = event.getPlayer();
        String normalized = normalizeName(player.getUsername());
        PlayerRecord record = playersByName.computeIfAbsent(normalized, ignored -> new PlayerRecord());

        record.name = player.getUsername();
        record.uuid = player.getUniqueId().toString();
        record.lastLoggedInAt = nowIso();
        record.currentServer = player.getCurrentServer()
                .map(c -> c.getServerInfo().getName())
                .orElse("");

        saveState();
        broadcastSnapshot();
    }

    @Subscribe
    public void onServerConnected(ServerConnectedEvent event) {
        Player player = event.getPlayer();
        String normalized = normalizeName(player.getUsername());
        PlayerRecord record = playersByName.computeIfAbsent(normalized, ignored -> new PlayerRecord());

        record.name = player.getUsername();
        record.uuid = player.getUniqueId().toString();
        record.currentServer = event.getServer().getServerInfo().getName();
        if (record.lastLoggedInAt == null || record.lastLoggedInAt.isBlank()) {
            record.lastLoggedInAt = nowIso();
        }

        saveState();
        broadcastSnapshot();
    }

    @Subscribe
    public void onDisconnect(DisconnectEvent event) {
        Player player = event.getPlayer();
        String normalized = normalizeName(player.getUsername());
        PlayerRecord record = playersByName.computeIfAbsent(normalized, ignored -> new PlayerRecord());

        record.name = player.getUsername();
        record.uuid = player.getUniqueId().toString();
        record.lastLoggedOutAt = nowIso();
        record.currentServer = "";

        saveState();
        broadcastSnapshot();
    }

    private PluginConfig loadConfig() throws IOException {
        Path configPath = dataDirectory.resolve(CONFIG_FILE);
        Properties properties = new Properties();

        if (!Files.exists(configPath)) {
            properties.setProperty("bindHost", "127.0.0.1");
            properties.setProperty("port", "19132");
            properties.setProperty("token", "");
            try (OutputStream out = Files.newOutputStream(configPath)) {
                properties.store(out, "SpoutMC Players Bridge config");
            }
        }

        try (InputStream inputStream = Files.newInputStream(configPath)) {
            properties.load(inputStream);
        }

        PluginConfig loaded = new PluginConfig();
        loaded.bindHost = properties.getProperty("bindHost", "127.0.0.1");
        loaded.port = Integer.parseInt(properties.getProperty("port", "19132"));
        loaded.token = properties.getProperty("token", "").trim();
        return loaded;
    }

    private void loadState() {
        Path path = dataDirectory.resolve(STATE_FILE);
        if (!Files.exists(path)) {
            return;
        }

        try (Reader reader = Files.newBufferedReader(path, StandardCharsets.UTF_8)) {
            PersistentState state = gson.fromJson(reader, PersistentState.class);
            if (state == null) {
                return;
            }
            if (state.playersByName != null) {
                playersByName.clear();
                playersByName.putAll(state.playersByName);
            }
            if (state.bansByName != null) {
                bansByName.clear();
                bansByName.putAll(state.bansByName);
            }
        } catch (Exception e) {
            logger.warn("[SpoutPlayers] Unable to load state file", e);
        }
    }

    private void saveState() {
        Path path = dataDirectory.resolve(STATE_FILE);
        PersistentState state = new PersistentState();
        state.playersByName = new ConcurrentHashMap<>(playersByName);
        state.bansByName = new ConcurrentHashMap<>(bansByName);

        try {
            Files.createDirectories(dataDirectory);
            Files.writeString(path, gson.toJson(state), StandardCharsets.UTF_8);
        } catch (Exception e) {
            logger.warn("[SpoutPlayers] Unable to save state file", e);
        }
    }

    private void seedOnlinePlayers() {
        for (Player player : proxy.getAllPlayers()) {
            String normalized = normalizeName(player.getUsername());
            PlayerRecord record = playersByName.computeIfAbsent(normalized, ignored -> new PlayerRecord());
            record.name = player.getUsername();
            record.uuid = player.getUniqueId().toString();
            if (record.lastLoggedInAt == null || record.lastLoggedInAt.isBlank()) {
                record.lastLoggedInAt = nowIso();
            }
            record.currentServer = player.getCurrentServer()
                    .map(c -> c.getServerInfo().getName())
                    .orElse("");
        }
        saveState();
    }

    private void startHttpServer() throws IOException {
        httpServer = HttpServer.create(new InetSocketAddress(config.bindHost, config.port), 0);
        httpServer.createContext("/", new ApiHandler());
        httpServer.setExecutor(Executors.newCachedThreadPool(runnable -> {
            Thread t = new Thread(runnable, "spoutmc-players-http");
            t.setUncaughtExceptionHandler((thread, throwable) ->
                    logger.error("[SpoutPlayers] Uncaught exception in HTTP thread {}", thread.getName(), throwable));
            return t;
        }));
        httpServer.start();
    }

    private void stopHttpServer() {
        if (httpServer != null) {
            httpServer.stop(0);
        }
        for (SseClient client : sseClients) {
            client.close();
        }
        sseClients.clear();
    }

    private List<PlayerView> snapshotPlayers() {
        refreshFromOnlinePlayers();

        List<PlayerView> players = new ArrayList<>();
        for (Map.Entry<String, PlayerRecord> entry : playersByName.entrySet()) {
            String normalized = entry.getKey();
            PlayerRecord record = entry.getValue();
            BanRecord ban = bansByName.get(normalized);

            PlayerView view = new PlayerView();
            view.name = record.name;
            view.avatarUrl = buildAvatarUrl(record.uuid);
            view.lastLoggedInAt = record.lastLoggedInAt;
            view.lastLoggedOutAt = record.lastLoggedOutAt;
            view.currentServer = record.currentServer;
            view.banned = ban != null;
            view.banReason = ban == null ? "" : ban.reason;

            if (view.banned) {
                view.status = "banned";
            } else if (view.currentServer != null && !view.currentServer.isBlank()) {
                view.status = "online";
            } else {
                view.status = "offline";
            }

            players.add(view);
        }

        players.sort(Comparator.comparing(p -> p.name == null ? "" : p.name.toLowerCase()));
        return players;
    }

    private void refreshFromOnlinePlayers() {
        for (Player player : proxy.getAllPlayers()) {
            String normalized = normalizeName(player.getUsername());
            PlayerRecord record = playersByName.computeIfAbsent(normalized, ignored -> new PlayerRecord());
            record.name = player.getUsername();
            record.uuid = player.getUniqueId().toString();
            if (record.lastLoggedInAt == null || record.lastLoggedInAt.isBlank()) {
                record.lastLoggedInAt = nowIso();
            }
            record.currentServer = player.getCurrentServer()
                    .map(c -> c.getServerInfo().getName())
                    .orElse("");
        }
    }

    private void broadcastSnapshot() {
        String payload = gson.toJson(snapshotPlayers());
        for (SseClient client : sseClients) {
            if (!client.sendSse(payload)) {
                sseClients.remove(client);
                client.close();
            }
        }
    }

    private String buildAvatarUrl(String uuid) {
        if (uuid == null || uuid.isBlank()) {
            return "";
        }
        return "https://crafatar.com/avatars/" + uuid + "?size=72&overlay";
    }

    private Optional<Player> findOnlinePlayer(String playerName) {
        return proxy.getAllPlayers()
                .stream()
                .filter(player -> player.getUsername().equalsIgnoreCase(playerName))
                .findFirst();
    }

    private String normalizeName(String playerName) {
        return playerName == null ? "" : playerName.toLowerCase();
    }

    private String nowIso() {
        return DateTimeFormatter.ISO_INSTANT.format(Instant.now());
    }

    private class ApiHandler implements HttpHandler {
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
                    sendJson(exchange, 200, gson.toJson(snapshotPlayers()));
                    return;
                }

                if ("GET".equalsIgnoreCase(method) && "/players/stream".equals(path)) {
                    handlePlayersStream(exchange);
                    return;
                }

                if ("POST".equalsIgnoreCase(method) && path.startsWith("/players/")) {
                    handlePlayerAction(exchange, path);
                    return;
                }

                sendJson(exchange, 404, "{\"error\":\"not found\"}");
            } catch (Throwable e) {
                logger.error("[SpoutPlayers] API handler error", e);
                try {
                    sendJson(exchange, 500, "{\"error\":\"internal server error\"}");
                } catch (Throwable writeError) {
                    logger.error("[SpoutPlayers] Failed to write error response", writeError);
                    try {
                        exchange.close();
                    } catch (Throwable ignored) {
                    }
                }
            }
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

            client.sendSse(gson.toJson(snapshotPlayers()));

            while (!client.closed) {
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

            if ("message".equals(action)) {
                String message = getJsonString(body, "message");
                if (message == null || message.isBlank()) {
                    sendJson(exchange, 400, "{\"error\":\"message is required\"}");
                    return;
                }

                Optional<Player> player = findOnlinePlayer(playerName);
                if (player.isEmpty()) {
                    sendJson(exchange, 404, "{\"error\":\"player not online\"}");
                    return;
                }

                player.get().sendMessage(Component.text(message));
                sendJson(exchange, 202, "{\"status\":\"message sent\"}");
                return;
            }

            if ("kick".equals(action)) {
                String reason = getJsonString(body, "reason");
                if (reason == null || reason.isBlank()) {
                    reason = "Kicked by admin";
                }

                Optional<Player> player = findOnlinePlayer(playerName);
                if (player.isEmpty()) {
                    sendJson(exchange, 404, "{\"error\":\"player not online\"}");
                    return;
                }

                player.get().disconnect(Component.text(reason));

                String normalized = normalizeName(playerName);
                PlayerRecord record = playersByName.computeIfAbsent(normalized, ignored -> new PlayerRecord());
                record.name = player.get().getUsername();
                record.uuid = player.get().getUniqueId().toString();
                record.lastLoggedOutAt = nowIso();
                record.currentServer = "";
                saveState();
                broadcastSnapshot();

                sendJson(exchange, 202, "{\"status\":\"player kicked\"}");
                return;
            }

            if ("ban".equals(action)) {
                String reason = getJsonString(body, "reason");
                if (reason == null || reason.isBlank()) {
                    reason = "Banned by admin";
                }

                String normalized = normalizeName(playerName);
                BanRecord banRecord = new BanRecord();
                banRecord.reason = reason;
                banRecord.bannedAt = nowIso();
                bansByName.put(normalized, banRecord);

                Optional<Player> player = findOnlinePlayer(playerName);
                if (player.isPresent()) {
                    player.get().disconnect(Component.text("You are banned: " + reason));

                    PlayerRecord record = playersByName.computeIfAbsent(normalized, ignored -> new PlayerRecord());
                    record.name = player.get().getUsername();
                    record.uuid = player.get().getUniqueId().toString();
                    record.lastLoggedOutAt = nowIso();
                    record.currentServer = "";
                } else {
                    PlayerRecord record = playersByName.computeIfAbsent(normalized, ignored -> new PlayerRecord());
                    if (record.name == null || record.name.isBlank()) {
                        record.name = playerName;
                    }
                    record.currentServer = "";
                }

                saveState();
                broadcastSnapshot();
                sendJson(exchange, 202, "{\"status\":\"player banned\"}");
                return;
            }

            if ("unban".equals(action)) {
                String normalized = normalizeName(playerName);
                BanRecord removed = bansByName.remove(normalized);
                if (removed == null) {
                    sendJson(exchange, 404, "{\"error\":\"player is not banned\"}");
                    return;
                }

                PlayerRecord record = playersByName.computeIfAbsent(normalized, ignored -> new PlayerRecord());
                if (record.name == null || record.name.isBlank()) {
                    record.name = playerName;
                }

                saveState();
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

    private static final class SseClient {
        private final HttpExchange exchange;
        private final OutputStream outputStream;
        private final Object lock = new Object();
        private volatile boolean closed;

        private SseClient(HttpExchange exchange) throws IOException {
            this.exchange = exchange;
            this.outputStream = exchange.getResponseBody();
            this.closed = false;
        }

        private boolean sendSse(String jsonPayload) {
            String line = "data: " + jsonPayload + "\n\n";
            return writeLine(line);
        }

        private boolean sendComment(String comment) {
            return writeLine(": " + comment + "\n\n");
        }

        private boolean writeLine(String payload) {
            synchronized (lock) {
                if (closed) {
                    return false;
                }
                try {
                    outputStream.write(payload.getBytes(StandardCharsets.UTF_8));
                    outputStream.flush();
                    return true;
                } catch (IOException ignored) {
                    close();
                    return false;
                }
            }
        }

        private void close() {
            synchronized (lock) {
                if (closed) {
                    return;
                }
                closed = true;
                try {
                    outputStream.close();
                } catch (IOException ignored) {
                }
                exchange.close();
            }
        }
    }

    private static final class PluginConfig {
        private String bindHost;
        private int port;
        private String token;
    }

    private static final class PlayerRecord {
        private String name;
        private String uuid;
        private String lastLoggedInAt;
        private String lastLoggedOutAt;
        private String currentServer;
    }

    private static final class BanRecord {
        private String reason;
        private String bannedAt;
    }

    private static final class PlayerView {
        private String name;
        private String avatarUrl;
        private String lastLoggedInAt;
        private String lastLoggedOutAt;
        private String currentServer;
        private boolean banned;
        private String banReason;
        private String status;
    }

    private static final class PersistentState {
        private Map<String, PlayerRecord> playersByName;
        private Map<String, BanRecord> bansByName;
    }
}
