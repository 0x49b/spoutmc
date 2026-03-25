package io.spoutmc.velocityplayers.model;

import java.util.Map;

public final class PersistentState {
    /**
     * Canonical persisted state keyed by UUID.
     * If this is null, we may be loading a legacy state.json that used name-keying.
     */
    public Map<String, PlayerRecord> playersByUUID;
    public Map<String, BanRecord> bansByUUID;

    /** Legacy state keyed by normalized player name. */
    public Map<String, PlayerRecord> playersByName;
    /** Legacy ban state keyed by normalized player name. */
    public Map<String, BanRecord> bansByName;
}
