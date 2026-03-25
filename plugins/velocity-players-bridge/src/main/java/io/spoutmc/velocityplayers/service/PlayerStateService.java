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
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;

public final class PlayerStateService {
    private static final String STATE_FILE = "state.json";

    private final ProxyServer proxy;
    private final Logger logger;
    private final Gson gson;
    private final Path dataDirectory;
    // Canonical persisted state keyed by UUID.
    private final Map<String, PlayerRecord> playersByUUID = new ConcurrentHashMap<>();
    private final Map<String, String> uuidByNormalizedName = new ConcurrentHashMap<>();
    private final Map<String, BanRecord> bansByUUID = new ConcurrentHashMap<>();

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

            playersByUUID.clear();
            uuidByNormalizedName.clear();
            bansByUUID.clear();

            // New schema (UUID-keyed).
            if (state.playersByUUID != null) {
                playersByUUID.putAll(state.playersByUUID);
                for (Map.Entry<String, PlayerRecord> e : playersByUUID.entrySet()) {
                    PlayerRecord r = e.getValue();
                    if (r == null) {
                        continue;
                    }
                    if (r.name != null && !r.name.isBlank()) {
                        uuidByNormalizedName.put(PluginUtils.normalizeName(r.name), e.getKey());
                    }
                }
            }

            // Legacy schema (name-keyed).
            if (playersByUUID.isEmpty() && state.playersByName != null) {
                for (Map.Entry<String, PlayerRecord> e : state.playersByName.entrySet()) {
                    PlayerRecord r = e.getValue();
                    if (r == null) {
                        continue;
                    }
                    String normalizedName = e.getKey();
                    String uuid = r.uuid;
                    if (uuid == null || uuid.isBlank()) {
                        continue;
                    }
                    r.uuid = uuid;
                    playersByUUID.put(uuid, r);
                    uuidByNormalizedName.put(normalizedName, uuid);
                }
            }

            // New bans (UUID-keyed).
            if (state.bansByUUID != null) {
                bansByUUID.putAll(state.bansByUUID);
            }

            // Legacy bans (name-keyed) migration.
            if (bansByUUID.isEmpty() && state.bansByName != null) {
                for (Map.Entry<String, BanRecord> e : state.bansByName.entrySet()) {
                    String normalizedName = e.getKey();
                    BanRecord ban = e.getValue();
                    if (ban == null) {
                        continue;
                    }
                    String uuid = uuidByNormalizedName.get(normalizedName);
                    if (uuid == null) {
                        continue;
                    }
                    bansByUUID.put(uuid, ban);
                }
            }
        } catch (Exception e) {
            logger.warn("[SpoutPlayers] Unable to load state file", e);
        }
    }

    public void saveState() {
        Path path = dataDirectory.resolve(STATE_FILE);
        PersistentState state = new PersistentState();
        state.playersByUUID = new ConcurrentHashMap<>(playersByUUID);
        state.bansByUUID = new ConcurrentHashMap<>(bansByUUID);

        try {
            Files.createDirectories(dataDirectory);
            Files.writeString(path, gson.toJson(state), StandardCharsets.UTF_8);
        } catch (Exception e) {
            logger.warn("[SpoutPlayers] Unable to save state file", e);
        }
    }

    public void seedOnlinePlayers() {
        for (Player player : proxy.getAllPlayers()) {
            String uuid = player.getUniqueId().toString();
            String normalized = PluginUtils.normalizeName(player.getUsername());
            PlayerRecord record = playersByUUID.computeIfAbsent(uuid, ignored -> new PlayerRecord());
            record.uuid = uuid;
            record.name = player.getUsername();
            if (record.lastLoggedInAt == null || record.lastLoggedInAt.isBlank()) {
                record.lastLoggedInAt = PluginUtils.nowIso();
            }
            record.currentServer = player.getCurrentServer()
                    .map(c -> c.getServerInfo().getName())
                    .orElse("");
            uuidByNormalizedName.put(normalized, uuid);
        }
        saveState();
    }

    public void onPostLogin(Player player) {
        String uuid = player.getUniqueId().toString();
        String normalized = PluginUtils.normalizeName(player.getUsername());
        PlayerRecord record = playersByUUID.computeIfAbsent(uuid, ignored -> new PlayerRecord());
        record.uuid = uuid;
        record.name = player.getUsername();
        record.lastLoggedInAt = PluginUtils.nowIso();
        record.currentServer = player.getCurrentServer()
                .map(c -> c.getServerInfo().getName())
                .orElse("");
        uuidByNormalizedName.put(normalized, uuid);
    }

    public void onServerConnected(Player player, String serverName) {
        String uuid = player.getUniqueId().toString();
        String normalized = PluginUtils.normalizeName(player.getUsername());
        PlayerRecord record = playersByUUID.computeIfAbsent(uuid, ignored -> new PlayerRecord());
        record.uuid = uuid;
        record.name = player.getUsername();
        record.currentServer = serverName;
        if (record.lastLoggedInAt == null || record.lastLoggedInAt.isBlank()) {
            record.lastLoggedInAt = PluginUtils.nowIso();
        }
        uuidByNormalizedName.put(normalized, uuid);
    }

    public void onDisconnect(Player player) {
        String uuid = player.getUniqueId().toString();
        String normalized = PluginUtils.normalizeName(player.getUsername());
        PlayerRecord record = playersByUUID.computeIfAbsent(uuid, ignored -> new PlayerRecord());
        record.uuid = uuid;
        record.name = player.getUsername();
        record.lastLoggedOutAt = PluginUtils.nowIso();
        record.currentServer = "";
        uuidByNormalizedName.put(normalized, uuid);
    }

    public void markPlayerDisconnected(Player player) {
        onDisconnect(player);
    }

    public List<PlayerView> snapshotPlayers() {
        refreshFromOnlinePlayers();

        List<PlayerView> players = new ArrayList<>();
        for (Map.Entry<String, PlayerRecord> entry : playersByUUID.entrySet()) {
            String uuid = entry.getKey();
            PlayerRecord record = entry.getValue();
            BanRecord ban = bansByUUID.get(uuid);

            PlayerView view = new PlayerView();
            view.name = record.name;
            view.uuid = uuid;
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
            String uuid = player.getUniqueId().toString();
            String normalized = PluginUtils.normalizeName(player.getUsername());
            PlayerRecord record = playersByUUID.computeIfAbsent(uuid, ignored -> new PlayerRecord());
            record.uuid = uuid;
            record.name = player.getUsername();
            if (record.lastLoggedInAt == null || record.lastLoggedInAt.isBlank()) {
                record.lastLoggedInAt = PluginUtils.nowIso();
            }
            record.currentServer = player.getCurrentServer()
                    .map(c -> c.getServerInfo().getName())
                    .orElse("");
            uuidByNormalizedName.put(normalized, uuid);
        }
    }

    private static boolean looksLikeUUID(String value) {
        if (value == null) {
            return false;
        }
        String trimmed = value.trim();
        if (trimmed.isEmpty()) {
            return false;
        }
        try {
            UUID.fromString(trimmed);
            return true;
        } catch (IllegalArgumentException ignored) {
            return false;
        }
    }

    private Optional<String> resolveUUID(String identifier) {
        if (identifier == null) {
            return Optional.empty();
        }
        if (looksLikeUUID(identifier)) {
            return Optional.of(identifier.trim());
        }
        String normalized = PluginUtils.normalizeName(identifier);
        return Optional.ofNullable(uuidByNormalizedName.get(normalized));
    }

    public Optional<Player> findOnlinePlayer(String identifier) {
        if (looksLikeUUID(identifier)) {
            String uuid = identifier.trim();
            return proxy.getAllPlayers()
                    .stream()
                    .filter(player -> player.getUniqueId().toString().equalsIgnoreCase(uuid))
                    .findFirst();
        }

        return proxy.getAllPlayers()
                .stream()
                .filter(player -> player.getUsername().equalsIgnoreCase(identifier))
                .findFirst();
    }

    public BanRecord getBan(String identifier) {
        Optional<String> uuid = resolveUUID(identifier);
        return uuid.map(bansByUUID::get).orElse(null);
    }

    public void banPlayer(String identifier, String reason) {
        Optional<String> uuid = resolveUUID(identifier);
        String finalUuid = uuid.orElse(null);
        if (finalUuid == null) {
            // Compatibility fallback: if identifier is a name and the player is online,
            // we can resolve the UUID from the online player.
            Optional<Player> online = findOnlinePlayer(identifier);
            if (online.isPresent()) {
                finalUuid = online.get().getUniqueId().toString();
                String normalized = PluginUtils.normalizeName(online.get().getUsername());
                uuidByNormalizedName.put(normalized, finalUuid);
                PlayerRecord record = playersByUUID.computeIfAbsent(finalUuid, ignored -> new PlayerRecord());
                record.uuid = finalUuid;
                record.name = online.get().getUsername();
            }
        }

        if (finalUuid == null || finalUuid.isBlank()) {
            // Can't persist a UUID-based ban without a UUID.
            return;
        }

        BanRecord banRecord = new BanRecord();
        banRecord.reason = reason;
        banRecord.bannedAt = PluginUtils.nowIso();
        bansByUUID.put(finalUuid, banRecord);
    }

    public boolean unbanPlayer(String identifier) {
        Optional<String> uuid = resolveUUID(identifier);
        if (uuid.isEmpty()) {
            // Compatibility fallback: if identifier is a name and player is online,
            // derive the UUID from the online player.
            Optional<Player> online = findOnlinePlayer(identifier);
            if (online.isPresent()) {
                uuid = Optional.of(online.get().getUniqueId().toString());
            }
        }

        if (uuid.isEmpty()) {
            return false;
        }

        BanRecord removed = bansByUUID.remove(uuid.get());
        return removed != null;
    }

    public PlayerRecord ensurePlayerRecord(String identifier) {
        Optional<String> uuid = resolveUUID(identifier);
        if (uuid.isEmpty()) {
            return new PlayerRecord();
        }

        PlayerRecord record = playersByUUID.computeIfAbsent(uuid.get(), ignored -> new PlayerRecord());
        record.uuid = uuid.get();
        if (!looksLikeUUID(identifier)) {
            // Best-effort name persistence for legacy calls / unknown online UUIDs.
            record.name = identifier;
            uuidByNormalizedName.put(PluginUtils.normalizeName(identifier), uuid.get());
        }
        return record;
    }
}
