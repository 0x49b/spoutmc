package io.spoutmc.velocityplayers.model;

import java.util.List;

public final class PlayerView {
    public String name;
    /** Canonical Minecraft UUID (stable across gamertag changes). */
    public String uuid;
    public String avatarUrl;
    public String lastLoggedInAt;
    public String lastLoggedOutAt;
    public String currentServer;
    public boolean banned;
    public String banReason;
    public String status;
    public String clientBrand;
    public List<String> clientMods;
}
