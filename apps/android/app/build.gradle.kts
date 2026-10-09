import org.jetbrains.kotlin.gradle.dsl.JvmTarget

// Developer installs have a separate identity from the eventual signed release channel.
plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
    id("org.jetbrains.kotlin.plugin.compose")
}
android {
    namespace = "io.github.lynchbros.alreadyon"
    compileSdk = 36
    defaultConfig {
        applicationId = "io.github.lynchbros.alreadyon"
        minSdk = 26
        targetSdk = 36
        versionCode = 19
        versionName = "0.1.18-dev"
    }
    buildTypes {
        debug { applicationIdSuffix = ".debug" }
        // Release signing is intentionally unconfigured until the owner supplies a key.
        release { isMinifyEnabled = false }
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    buildFeatures { compose = true }
}
kotlin { compilerOptions { jvmTarget.set(JvmTarget.JVM_17) } }

dependencies {
    implementation(platform("androidx.compose:compose-bom:2025.12.00"))
    implementation("androidx.activity:activity-compose:1.12.4")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.foundation:foundation")
    implementation("androidx.compose.ui:ui")
    implementation("androidx.lifecycle:lifecycle-viewmodel-compose:2.9.4")
    implementation("androidx.lifecycle:lifecycle-runtime-compose:2.9.4")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.10.2")
    implementation("com.squareup.okhttp3:okhttp:4.12.0")
    implementation("androidx.media3:media3-exoplayer:1.9.4")
    implementation("androidx.media3:media3-exoplayer-hls:1.9.4")
    implementation("androidx.media3:media3-ui-compose:1.9.4")
    implementation("androidx.media3:media3-datasource-okhttp:1.9.4")
}
