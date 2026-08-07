import java.util.Properties

plugins {
    id("com.android.application")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
}

// Firebase (FCM) native wiring. google-services.json comes from the Firebase
// console (ops step — see DEVOPS-TODO.md); the google-services plugin hard-fails
// the build when the file is missing, so only apply it once it exists. Until
// then push still works via FlutterFire's programmatic init (firebase_options.dart).
if (file("google-services.json").exists()) {
    pluginManager.apply("com.google.gms.google-services")
} else {
    logger.warn(
        "google-services.json not found in android/app — skipping the " +
            "com.google.gms.google-services plugin (FCM uses the programmatic FlutterFire init)."
    )
}

// Release signing: the upload keystore path + passwords live in the gitignored
// android/key.properties (see DEPLOY-CREDS.local.md). Debug builds don't need it.
val keystoreProperties = Properties()
val keystorePropertiesFile = rootProject.file("key.properties")
if (keystorePropertiesFile.exists()) {
    keystorePropertiesFile.inputStream().use { keystoreProperties.load(it) }
} else {
    logger.warn(
        "android/key.properties not found — release builds will fail; see DEPLOY-CREDS.local.md"
    )
}

android {
    namespace = "com.salehcard.salehcard_app"
    compileSdk = flutter.compileSdkVersion
    ndkVersion = flutter.ndkVersion

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    defaultConfig {
        // Play Console app is registered under this package name. Kept
        // distinct from `namespace` (which still owns the R/MainActivity
        // classes) so only the store-facing package id changes.
        // NOTE: flashcash.global was already uploaded (and rejected) under a
        // discarded Play developer account, which permanently claimed that
        // package name — Google never releases it back to the pool. This app
        // is submitted under a new org account, hence the new package id.
        applicationId = "com.flashcashglobal.app"
        // You can update the following values to match your application needs.
        // For more information, see: https://flutter.dev/to/review-gradle-config.
        minSdk = flutter.minSdkVersion
        targetSdk = flutter.targetSdkVersion
        versionCode = flutter.versionCode
        versionName = flutter.versionName
    }

    signingConfigs {
        create("release") {
            keyAlias = keystoreProperties["keyAlias"] as String?
            keyPassword = keystoreProperties["keyPassword"] as String?
            storeFile = (keystoreProperties["storeFile"] as String?)?.let { file(it) }
            storePassword = keystoreProperties["storePassword"] as String?
        }
    }

    buildTypes {
        release {
            // No debug fallback: without key.properties the storeFile stays null
            // and the release build fails loudly at :app:validateSigningRelease.
            signingConfig = signingConfigs.getByName("release")
        }
    }
}

kotlin {
    compilerOptions {
        jvmTarget = org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17
    }
}

flutter {
    source = "../.."
}
