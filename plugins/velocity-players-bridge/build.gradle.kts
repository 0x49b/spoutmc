plugins {
    java
    id("com.gradleup.shadow") version "8.3.0"
}

group = "io.spoutmc"
version = "0.1.0"

java {
    sourceCompatibility = JavaVersion.VERSION_17
    targetCompatibility = JavaVersion.VERSION_17
}

repositories {
    mavenCentral()
    maven("https://repo.papermc.io/repository/maven-public/")
}

dependencies {
    compileOnly("com.velocitypowered:velocity-api:3.3.0-SNAPSHOT")
    annotationProcessor("com.velocitypowered:velocity-api:3.3.0-SNAPSHOT")
    implementation("com.google.code.gson:gson:2.11.0")
}

tasks {
    compileJava {
        // Allow using newer installed JDKs while producing Java 17 bytecode.
        options.release.set(17)
    }

    shadowJar {
        archiveClassifier.set("")
    }
}
