package io.spoutmc.velocityplayers.service;

import com.google.gson.Gson;
import com.velocitypowered.api.proxy.Player;
import com.velocitypowered.api.proxy.ProxyServer;
import io.spoutmc.velocityplayers.model.BanRecord;
import io.spoutmc.velocityplayers.model.PersistentState;
import io.spoutmc.velocityplayers.model.PlayerRecord;
import io.spoutmc.velocityplayers.model.PlayerView;
import io.spoutmc.velocityplayers.util.PluginUtils;
import org.slf4j.Logger;

import java.io.Reader;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.concurrent.ConcurrentHashMap;

public final class PlayerStateService {
    private static final String STATE_FILE = "state.json";

    private final ProxyServer proxy;
    private final Logger logger;
    private final Gson gson;
    private final Path dataDirectory;
    private final Map<String, PlayerRecord> playersByName = new ConcurrentHashMap<>();
    private final Map<String, BanRecord> bansByName = new ConcurrentHashMap<>();

    public PlayerStateService(ProxyServer proxy, Logger logger, Gson gson, Path dataDirectory) {
        this.proxy = proxy;
        this.logger = logger;
        this.gson = gson;
        this.dataDirectory = dataDirectory;
    }

    public void loadState() {
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

    public void saveState() {
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

    public void seedOnlinePlayers() {
        for (Player player : proxy.getAllPlayers()) {
            String normalized = PluginUtils.normalizeName(player.getUsername());
            PlayerRecord record = playersByName.computeIfAbsent(normalized, ignored -> new PlayerRecord());
            record.name = player.getUsername();
            record.uuid = player.getUniqueId().toString();
            if (record.lastLoggedInAt == null || record.lastLoggedInAt.isBlank()) {
                record.lastLoggedInAt = PluginUtils.nowIso();
            }
            record.currentServer = player.getCurrentServer()
                    .map(c -> c.getServerInfo().getName())
                    .orElse("");
        }
        saveState();
    }

    public void onPostLogin(Player player) {
        String normalized = PluginUtils.normalizeName(player.getUsername());
        PlayerRecord record = playersByName.computeIfAbsent(normalized, ignored -> new PlayerRecord());
        record.name = player.getUsername();
        record.uuid = player.getUniqueId().toString();
        record.lastLoggedInAt = PluginUtils.nowIso();
        record.currentServer = player.getCurrentServer()
                .map(c -> c.getServerInfo().getName())
                .orElse("");
    }

    public void onServerConnected(Player player, String serverName) {
        String normalized = PluginUtils.normalizeName(player.getUsername());
        PlayerRecord record = playersByName.computeIfAbsent(normalized, ignored -> new PlayerRecord());
        record.name = player.getUsername();
        record.uuid = player.getUniqueId().toString();
        record.currentServer = serverName;
        if (record.lastLoggedInAt == null || record.lastLoggedInAt.isBlank()) {
            record.lastLoggedInAt = PluginUtils.nowIso();
        }
    }

    public void onDisconnect(Player player) {
        String normalized = PluginUtils.normalizeName(player.getUsername());
        PlayerRecord record = playersByName.computeIfAbsent(normalized, ignored -> new PlayerRecord());
        record.name = player.getUsername();
        record.uuid = player.getUniqueId().toString();
        record.lastLoggedOutAt = PluginUtils.nowIso();
        record.currentServer = "";
    }

    public void markPlayerDisconnected(Player player) {
        onDisconnect(player);
    }

    public List<PlayerView> snapshotPlayers() {
        refreshFromOnlinePlayers();

        List<PlayerView> players = new ArrayList<>();
        for (Map.Entry<String, PlayerRecord> entry : playersByName.entrySet()) {
            String normalized = entry.getKey();
            PlayerRecord record = entry.getValue();
            BanRecord ban = bansByName.get(normalized);

            PlayerView view = new PlayerView();
            view.name = record.name;
            view.avatarUrl = PluginUtils.buildAvatarUrl(record.uuid);
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
            String normalized = PluginUtils.normalizeName(player.getUsername());
            PlayerRecord record = playersByName.computeIfAbsent(normalized, ignored -> new PlayerRecord());
            record.name = player.getUsername();
            record.uuid = player.getUniqueId().toString();
            if (record.lastLoggedInAt == null || record.lastLoggedInAt.isBlank()) {
                record.lastLoggedInAt = PluginUtils.nowIso();
            }
            record.currentServer = player.getCurrentServer()
                    .map(c -> c.getServerInfo().getName())
                    .orElse("");
        }
    }

    public Optional<Player> findOnlinePlayer(String playerName) {
        return proxy.getAllPlayers()
                .stream()
                .filter(player -> player.getUsername().equalsIgnoreCase(playerName))
                .findFirst();
    }

    public BanRecord getBan(String playerName) {
        return bansByName.get(PluginUtils.normalizeName(playerName));
    }

    public void banPlayer(String playerName, String reason) {
        String normalized = PluginUtils.normalizeName(playerName);
        BanRecord banRecord = new BanRecord();
        banRecord.reason = reason;
        banRecord.bannedAt = PluginUtils.nowIso();
        bansByName.put(normalized, banRecord);
    }

    public boolean unbanPlayer(String playerName) {
        String normalized = PluginUtils.normalizeName(playerName);
        BanRecord removed = bansByName.remove(normalized);
        return removed != null;
    }

    public PlayerRecord ensurePlayerRecord(String playerName) {
        String normalized = PluginUtils.normalizeName(playerName);
        return playersByName.computeIfAbsent(normalized, ignored -> new PlayerRecord());
    }
}
