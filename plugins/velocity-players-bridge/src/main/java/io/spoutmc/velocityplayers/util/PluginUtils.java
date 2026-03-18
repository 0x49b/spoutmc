package io.spoutmc.velocityplayers.util;

import java.time.Instant;
import java.time.format.DateTimeFormatter;

public final class PluginUtils {
    private PluginUtils() {
    }

    public static String normalizeName(String playerName) {
        return playerName == null ? "" : playerName.toLowerCase();
    }

    public static String nowIso() {
        return DateTimeFormatter.ISO_INSTANT.format(Instant.now());
    }

    public static String buildAvatarUrl(String uuid) {
        if (uuid == null || uuid.isBlank()) {
            return "";
        }
        return "https://crafatar.com/avatars/" + uuid + "?size=72&overlay";
    }
}
