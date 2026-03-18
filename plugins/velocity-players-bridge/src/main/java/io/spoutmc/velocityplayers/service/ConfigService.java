package io.spoutmc.velocityplayers.service;

import io.spoutmc.velocityplayers.model.PluginConfig;

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Properties;

public final class ConfigService {
    private static final String CONFIG_FILE = "config.properties";

    private final Path dataDirectory;

    public ConfigService(Path dataDirectory) {
        this.dataDirectory = dataDirectory;
    }

    public PluginConfig loadConfig() throws IOException {
        Path configPath = dataDirectory.resolve(CONFIG_FILE);
        Properties properties = new Properties();

        if (!Files.exists(configPath)) {
            properties.setProperty("bindHost", "0.0.0.0");
            properties.setProperty("port", "29132");
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
}
