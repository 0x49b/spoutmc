package io.spoutmc.velocityplayers.service;

import com.google.gson.Gson;
import com.google.gson.JsonObject;
import com.velocitypowered.api.proxy.Player;
import io.spoutmc.velocityplayers.model.ChatMessageRecord;
import io.spoutmc.velocityplayers.model.PluginConfig;
import io.spoutmc.velocityplayers.util.PluginUtils;
import net.kyori.adventure.text.Component;
import net.kyori.adventure.text.format.NamedTextColor;
import org.slf4j.Logger;

import java.io.IOException;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.LinkedHashSet;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

public final class ChatService {
    private static final Pattern PRIVATE_MESSAGE_COMMAND_PATTERN = Pattern.compile("^(msg|tell|w|whisper|m)\\s+(\\S+)\\s+(.+)$", Pattern.CASE_INSENSITIVE);
    private static final Pattern FORWARDING_SECRET_FILE_PATTERN = Pattern.compile("^\\s*forwarding-secret-file\\s*=\\s*\"([^\"]+)\"\\s*$");

    private final Logger logger;
    private final PluginConfig config;
    private final String chatIngestSecret;
    private final Gson gson = new Gson();

    private final Map<String, List<ChatMessageRecord>> chatByThread = new ConcurrentHashMap<>();
    /** Last staff user id that messaged this MC player (normalized name) — /r routes here */
    private final Map<String, Long> lastStaffByPlayer = new ConcurrentHashMap<>();
    private final Set<String> webReplyHandles = Set.of("admin", "mod", "moderator", "editor", "staff");

    public ChatService(Logger logger, PluginConfig config, Path dataDirectory) {
        this.logger = logger;
        this.config = config;
        this.chatIngestSecret = resolveForwardingSecret(dataDirectory);
    }

    private static String threadKey(String normalizedPlayer, long staffUserId) {
        return normalizedPlayer + ":" + staffUserId;
    }

    public boolean handlePrivateMessageReplyAlias(Player player, String command) {
        if (command == null || command.isBlank()) {
            return false;
        }

        Matcher matcher = PRIVATE_MESSAGE_COMMAND_PATTERN.matcher(command.trim());
        if (!matcher.matches()) {
            return false;
        }

        String target = matcher.group(2);
        String message = matcher.group(3);
        if (target == null || message == null || message.isBlank()) {
            return false;
        }
        if (!isWebReplyTarget(target)) {
            return false;
        }

        if (captureIncomingReply(player, message)) {
            sendPlayerReplyFeedback(player, message.trim());
            return true;
        }
        return false;
    }

    public List<ChatMessageRecord> getChat(String playerName, long staffUserId) {
        String normalized = PluginUtils.normalizeName(playerName).trim();
        String key = threadKey(normalized, staffUserId);
        List<ChatMessageRecord> messages = chatByThread.getOrDefault(key, new ArrayList<>());
        synchronized (messages) {
            return new ArrayList<>(messages);
        }
    }

    public void sendStaffMessage(Player player, long staffUserId, String sender, String role, String message, boolean newConversation) {
        String senderLabel = sender == null || sender.isBlank() ? "SpoutMC" : sender.trim();
        String roleLabel = role == null || role.isBlank() ? "Staff" : role.trim();
        String normalized = PluginUtils.normalizeName(player.getUsername()).trim();
        String tKey = threadKey(normalized, staffUserId);
        List<ChatMessageRecord> chatMessages = chatByThread.computeIfAbsent(tKey, ignored -> new ArrayList<>());
        boolean firstInThread;
        synchronized (chatMessages) {
            firstInThread = chatMessages.stream().noneMatch(m -> "outgoing".equals(m.direction));
        }

        if (newConversation || firstInThread) {
            player.sendMessage(Component.text(
                    "A staff member is contacting you privately through SpoutMC support chat.",
                    NamedTextColor.RED
            ));
            player.sendMessage(Component.text(
                    "You can reply with /r <message> or /reply <message>.",
                    NamedTextColor.RED
            ));
        }

        ChatMessageRecord entry = new ChatMessageRecord();
        entry.direction = "outgoing";
        entry.player = player.getUsername();
        entry.staffUserId = staffUserId;
        entry.sender = senderLabel;
        entry.role = roleLabel;
        entry.message = message.trim();
        entry.timestamp = PluginUtils.nowIso();
        synchronized (chatMessages) {
            chatMessages.add(entry);
            trimChatHistory(chatMessages);
        }

        lastStaffByPlayer.put(normalized, staffUserId);

        String formattedMessage = "[" + roleLabel + "] " + senderLabel + ": " + message.trim();
        player.sendMessage(Component.text(formattedMessage, NamedTextColor.RED));
    }

    public boolean captureIncomingReply(Player player, String rawMessage) {
        String message = rawMessage == null ? "" : rawMessage.trim();
        if (message.isBlank()) {
            return false;
        }

        String normalizedPlayerName = PluginUtils.normalizeName(player.getUsername()).trim();
        Long staffId = lastStaffByPlayer.get(normalizedPlayerName);
        if (staffId == null || staffId <= 0) {
            return false;
        }

        String tKey = threadKey(normalizedPlayerName, staffId);
        List<ChatMessageRecord> chatMessages = chatByThread.get(tKey);
        if (chatMessages == null) {
            return false;
        }

        String timestampIso = PluginUtils.nowIso();
        synchronized (chatMessages) {
            if (chatMessages.stream().noneMatch(m -> "outgoing".equals(m.direction))) {
                return false;
            }

            ChatMessageRecord entry = new ChatMessageRecord();
            entry.direction = "incoming";
            entry.player = player.getUsername();
            entry.staffUserId = staffId;
            entry.sender = player.getUsername();
            entry.role = "";
            entry.message = message;
            entry.timestamp = timestampIso;
            chatMessages.add(entry);
            trimChatHistory(chatMessages);
        }

        ingestIncomingReply(player.getUsername(), player.getUniqueId().toString(), staffId, message, timestampIso);
        return true;
    }

    private void ingestIncomingReply(String mcPlayerName, String mcPlayerUuid, long staffUserId, String message, String timestampIso) {
        String urlStr = config.spoutmcChatIngestUrl;
        String secret = chatIngestSecret;
        if (urlStr == null || urlStr.isBlank() || secret == null || secret.isBlank()) {
            return;
        }

        Thread thread = new Thread(() -> postIngest(urlStr, secret, mcPlayerName, mcPlayerUuid, staffUserId, message, timestampIso), "spoutmc-chat-ingest");
        thread.setDaemon(true);
        thread.start();
    }

    private void postIngest(String urlStr, String secret, String mcPlayerName, String mcPlayerUuid, long staffUserId, String message, String timestampIso) {
        try {
            JsonObject body = new JsonObject();
            body.addProperty("playerName", mcPlayerName);
            body.addProperty("playerUuid", mcPlayerUuid);
            body.addProperty("staffUserId", staffUserId);
            body.addProperty("message", message);
            body.addProperty("timestamp", timestampIso);

            byte[] bytes = gson.toJson(body).getBytes(StandardCharsets.UTF_8);
            java.net.URL url = java.net.URI.create(urlStr).toURL();
            HttpURLConnection conn = (HttpURLConnection) url.openConnection();
            conn.setRequestMethod("POST");
            conn.setConnectTimeout(5000);
            conn.setReadTimeout(8000);
            conn.setDoOutput(true);
            conn.setRequestProperty("Content-Type", "application/json; charset=utf-8");
            conn.setRequestProperty("X-Spout-Chat-Ingest", secret);
            try (OutputStream out = conn.getOutputStream()) {
                out.write(bytes);
            }
            int code = conn.getResponseCode();
            if (code < 200 || code >= 300) {
                logger.warn("[SpoutPlayers] Chat ingest failed HTTP {}", code);
            }
            conn.disconnect();
        } catch (Exception e) {
            logger.warn("[SpoutPlayers] Chat ingest error: {}", e.toString());
        }
    }

    public void sendPlayerReplyFeedback(Player player, String message) {
        player.sendMessage(Component.text("Reply delivered to staff panel.", NamedTextColor.GRAY));
        player.sendMessage(Component.text("You -> Staff: " + message, NamedTextColor.GRAY));
    }

    private boolean isWebReplyTarget(String targetName) {
        if (targetName == null || targetName.isBlank()) {
            return false;
        }
        return webReplyHandles.contains(targetName.trim().toLowerCase());
    }

    private void trimChatHistory(List<ChatMessageRecord> chatMessages) {
        final int maxMessages = 200;
        if (chatMessages.size() <= maxMessages) {
            return;
        }
        chatMessages.subList(0, chatMessages.size() - maxMessages).clear();
    }

    private String resolveForwardingSecret(Path dataDirectory) {
        LinkedHashSet<Path> proxyRootCandidates = new LinkedHashSet<>();
        if (dataDirectory != null) {
            Path pluginsDir = dataDirectory.getParent();
            Path proxyRoot = pluginsDir == null ? null : pluginsDir.getParent();
            if (proxyRoot != null) {
                proxyRootCandidates.add(proxyRoot);
            }
        }

        String userDir = System.getProperty("user.dir");
        if (userDir != null && !userDir.isBlank()) {
            proxyRootCandidates.add(Paths.get(userDir));
        }

        for (Path proxyRoot : proxyRootCandidates) {
            String secret = readSecretFromVelocityToml(proxyRoot);
            if (secret != null && !secret.isBlank()) {
                logger.info("[SpoutPlayers] Using forwarding secret from velocity.toml path {}", proxyRoot.resolve("velocity.toml"));
                return secret;
            }
        }

        for (Path proxyRoot : proxyRootCandidates) {
            String secret = readSecretFile(proxyRoot.resolve("forwarding.secret"));
            if (secret != null && !secret.isBlank()) {
                logger.info("[SpoutPlayers] Using forwarding secret from fallback path {}", proxyRoot.resolve("forwarding.secret"));
                return secret;
            }
        }

        logger.warn("[SpoutPlayers] Could not resolve forwarding.secret for chat ingest; incoming /reply messages will not be persisted to SpoutMC");
        return "";
    }

    private String readSecretFromVelocityToml(Path proxyRoot) {
        Path velocityToml = proxyRoot.resolve("velocity.toml");
        if (!Files.exists(velocityToml)) {
            return "";
        }

        try {
            List<String> lines = Files.readAllLines(velocityToml, StandardCharsets.UTF_8);
            for (String line : lines) {
                Matcher matcher = FORWARDING_SECRET_FILE_PATTERN.matcher(line);
                if (!matcher.matches()) {
                    continue;
                }
                String configured = matcher.group(1).trim();
                if (configured.isEmpty()) {
                    continue;
                }

                Path configuredPath = Paths.get(configured);
                Path secretPath = configuredPath.isAbsolute() ? configuredPath : proxyRoot.resolve(configuredPath);
                String secret = readSecretFile(secretPath);
                if (secret != null && !secret.isBlank()) {
                    return secret;
                }
            }
        } catch (IOException e) {
            logger.warn("[SpoutPlayers] Failed reading velocity.toml for forwarding secret: {}", e.toString());
        }

        return "";
    }

    private String readSecretFile(Path path) {
        try {
            if (!Files.exists(path)) {
                return "";
            }
            return Files.readString(path, StandardCharsets.UTF_8).trim();
        } catch (IOException e) {
            logger.warn("[SpoutPlayers] Failed reading forwarding secret file {}: {}", path, e.toString());
            return "";
        }
    }
}
