plugins {
    java
    id("com.gradleup.shadow") version "8.3.0"
    `maven-publish`
}

group = "io.spoutmc"
version = (findProperty("releaseVersion") as String?) ?: "0.1.0"

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

publishing {
    publications {
        register<MavenPublication>("maven") {
            groupId = project.group.toString()
            artifactId = "velocity-players-bridge"
            version = project.version.toString()
            artifact(tasks.shadowJar.get())
        }
    }
    repositories {
        maven {
            name = "GitHubPackages"
            url = uri(
                System.getenv("GITHUB_REPOSITORY")?.let { "https://maven.pkg.github.com/$it" }
                    ?: (findProperty("gpr.repositoryUrl") as String? ?: "https://maven.pkg.github.com/OWNER/REPO"),
            )
            credentials {
                // Prefer non-blank gpr.*; empty properties must not win over env (would cause HTTP 401 on publish).
                username = sequenceOf(
                    findProperty("gpr.user") as String?,
                    System.getenv("GITHUB_ACTOR"),
                ).firstOrNull { !it.isNullOrBlank() } ?: ""
                password = sequenceOf(
                    findProperty("gpr.key") as String?,
                    System.getenv("GITHUB_TOKEN"),
                ).firstOrNull { !it.isNullOrBlank() } ?: ""
            }
        }
    }
}
