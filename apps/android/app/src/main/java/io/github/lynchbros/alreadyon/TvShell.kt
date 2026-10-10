package io.github.lynchbros.alreadyon

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.focusable
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusDirection
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.input.key.*
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import kotlinx.coroutines.delay

/** TV navigation is fixed-position presentation; tuning never rotates or remounts the shell. */
@Composable
internal fun TvShell(state: PhoneState, model: PhoneModel) {
    var fullscreen by rememberSaveable { mutableStateOf(false) }
    var controlsSignal by remember { mutableIntStateOf(0) }
    var controlsVisible by remember { mutableStateOf(false) }
    val focus = remember { FocusRequester() }
    val manager = LocalFocusManager.current
    val owner = LocalLifecycleOwner.current
    DisposableEffect(owner, model) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_STOP) model.suspendWatching()
            if (event == Lifecycle.Event.ON_START) model.resumeWatching()
        }
        owner.lifecycle.addObserver(observer)
        onDispose { owner.lifecycle.removeObserver(observer) }
    }
    LaunchedEffect(state.profile == null) { if (state.profile == null) fullscreen = false }
    LaunchedEffect(fullscreen) { if (fullscreen) focus.requestFocus() }
    LaunchedEffect(state.profile?.id) {
        while (state.profile != null) { delay(30_000); model.refreshGuide() }
    }
    BackHandler(enabled = fullscreen || state.settings != null || state.profile != null) {
        when {
            fullscreen -> { fullscreen = false; manager.clearFocus() }
            state.settings != null -> model.closeSettings()
            else -> model.changeProfile()
        }
    }
    PhoneTheme {
        val palette = phonePalette
        Surface(Modifier.fillMaxSize(), color = palette.background, contentColor = palette.text) {
        Column(Modifier.fillMaxSize().padding(if (fullscreen) 0.dp else 24.dp)
            .onPreviewKeyEvent { event ->
                if (!fullscreen) return@onPreviewKeyEvent false
                when (event.key) {
                    Key.DirectionLeft, Key.DirectionRight -> {
                        if (controlsVisible) return@onPreviewKeyEvent false
                        if (event.type == KeyEventType.KeyUp) model.stepChannel(if (event.key == Key.DirectionRight) 1 else -1)
                        true
                    }
                    Key.DirectionUp, Key.DirectionDown -> {
                        if (event.type == KeyEventType.KeyDown) {
                            if (!controlsVisible) controlsSignal++
                            else manager.moveFocus(if (event.key == Key.DirectionDown) FocusDirection.Next else FocusDirection.Previous)
                        }
                        true
                    }
                    else -> false
                }
            // Only fullscreen needs a focus target to receive remote keys with hidden controls.
            // In the guide and sign-in screens, focus belongs to the actionable children.
            }.then(if (fullscreen) Modifier.focusRequester(focus).focusable() else Modifier),
            verticalArrangement = Arrangement.spacedBy(if (fullscreen) 0.dp else 12.dp)) {
            if (!fullscreen) Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.SpaceBetween) {
                PhoneBrand()
                if (state.profile != null) ProfileMenu(state, model)
            }
            if (!fullscreen) {
                if (state.signedIn) state.error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
                if (state.busy) LinearProgressIndicator(Modifier.fillMaxWidth())
            }
            when {
                !state.signedIn -> ServerConnectScreen(state, model, Modifier.weight(1f))
                state.mustChangePassword -> AccountScreen(state, model, forced = true)
                state.settings == "account" -> AccountScreen(state, model)
                state.settings == "profile" -> ProfileSettingsScreen(state, model)
                state.profile == null -> ProfileScreen(state, model, Modifier.weight(1f))
                else -> {
                    if (state.channel != null) {
                        // One player call site keeps the stream/settings alive when returning to the guide.
                        Row(if (fullscreen) Modifier.fillMaxWidth().weight(1f) else Modifier.fillMaxWidth().height(180.dp),
                            horizontalArrangement = Arrangement.spacedBy(if (fullscreen) 0.dp else 20.dp)) {
                            PhonePlayer(state, model,
                                if (fullscreen) Modifier.fillMaxSize() else Modifier.width(320.dp).fillMaxHeight(),
                                fullscreen = fullscreen, tv = true, controlsSignal = controlsSignal,
                                onControlsVisibility = { controlsVisible = it }, onFullscreen = { if (!fullscreen) model.unmute(); fullscreen = !fullscreen })
                            if (!fullscreen) Column(Modifier.weight(1f).padding(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                                Text("${state.channel.number} · ${state.channel.name}", style = MaterialTheme.typography.titleLarge)
                                state.playback?.let { playback ->
                                    if (playback.seriesTitle.isNotBlank()) Text(playback.seriesTitle, style = MaterialTheme.typography.titleMedium)
                                    Text(playback.episodeLabel + playback.title, style = MaterialTheme.typography.titleMedium)
                                }
                                Text("Select a channel to watch fullscreen. Back returns to the guide.", color = palette.muted)
                            }
                        }
                    }
                    if (!fullscreen) ChannelGuide(state, model, palette, Modifier.weight(1f)) { channel ->
                        model.unmute(); model.watch(channel); fullscreen = true
                    }
                }
            }
        }
        }
    }
}
