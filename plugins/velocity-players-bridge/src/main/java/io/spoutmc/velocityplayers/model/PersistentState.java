package io.spoutmc.velocityplayers.model;

import java.util.Map;

public final class PersistentState {
    public Map<String, PlayerRecord> playersByName;
    public Map<String, BanRecord> bansByName;
}
