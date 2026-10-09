package io.github.lynchbros.alreadyon

import androidx.compose.foundation.clickable
import androidx.compose.foundation.Image
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.layout.ContentScale
import android.app.PictureInPictureParams
import android.os.Build
import android.util.Rational
import android.os.Bundle
import android.util.TypedValue
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import android.content.res.Configuration
import androidx.activity.ComponentActivity
import androidx.activity.compose.BackHandler
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.border
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Modifier
import androidx.compose.ui.Alignment
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.graphics.vector.PathParser
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.WindowInsetsControllerCompat
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import kotlinx.coroutines.delay
import okhttp3.HttpUrl.Companion.toHttpUrl
import java.net.URI
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale

/** Native phone shell; credentials exist only while the sign-in form is on screen. */
class MainActivity : ComponentActivity() {
    private var pip by mutableStateOf(false)
    private var pipEligible = false
    private val pipSupported get() = packageManager.hasSystemFeature(android.content.pm.PackageManager.FEATURE_PICTURE_IN_PICTURE)
    override fun onPictureInPictureModeChanged(isInPictureInPictureMode: Boolean, newConfig: Configuration) {
        super.onPictureInPictureModeChanged(isInPictureInPictureMode, newConfig)
        pip = isInPictureInPictureMode
    }
    override fun onUserLeaveHint() {
        super.onUserLeaveHint()
        if (pipSupported && pipEligible && Build.VERSION.SDK_INT < 31) enterPictureInPictureMode(PictureInPictureParams.Builder().setAspectRatio(Rational(16, 9)).build())
    }
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            val model: PhoneModel = viewModel()
            val state by model.state.collectAsStateWithLifecycle()
            val owner = LocalLifecycleOwner.current
            var fullscreen by rememberSaveable { mutableStateOf(false) }
            val orientation = LocalConfiguration.current.orientation
            LaunchedEffect(orientation) {
                if (state.channel != null) {
                    fullscreen = orientation == Configuration.ORIENTATION_LANDSCAPE
                }
            }
            // Rotation/fullscreen/PiP change presentation only; lifecycle resume handles a released lease.
            LaunchedEffect(state.channel == null) { if (state.channel == null) fullscreen = false }
            DisposableEffect(owner, model) {
                val observer = LifecycleEventObserver { _, event ->
                    when (event) {
                        Lifecycle.Event.ON_STOP -> model.suspendWatching()
                        Lifecycle.Event.ON_START -> model.resumeWatching()
                        else -> Unit
                    }
                }
                owner.lifecycle.addObserver(observer)
                onDispose { owner.lifecycle.removeObserver(observer) }
            }
            LaunchedEffect(state.playback != null, state.paused) {
                pipEligible = state.playback != null && !state.paused
                if (pipSupported && Build.VERSION.SDK_INT >= 31) setPictureInPictureParams(PictureInPictureParams.Builder()
                    .setAspectRatio(Rational(16, 9)).setAutoEnterEnabled(pipEligible).build())
            }
            val expanded = fullscreen || pip
            val view = LocalView.current
            DisposableEffect(fullscreen, view) {
                val controller = WindowCompat.getInsetsController(window, view)
                controller.systemBarsBehavior = WindowInsetsControllerCompat.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE
                if (fullscreen) controller.hide(WindowInsetsCompat.Type.systemBars())
                else controller.show(WindowInsetsCompat.Type.systemBars())
                onDispose { controller.show(WindowInsetsCompat.Type.systemBars()) }
            }
            BackHandler(enabled = fullscreen || state.settings != null || state.channel != null || state.profile != null) {
                when {
                    state.settings != null -> model.closeSettings()
                    fullscreen -> fullscreen = false
                    state.channel != null -> model.changeProfile()
                    else -> model.changeProfile()
                }
            }
            LaunchedEffect(state.profile?.id) {
                while (state.profile != null) { delay(30_000); model.refreshGuide() }
            }
            PhoneTheme {
                val palette = phonePalette
                val choosingProfile = state.signedIn && !state.mustChangePassword && state.profile == null
                Scaffold(contentWindowInsets = if (expanded) WindowInsets(0, 0, 0, 0) else WindowInsets.safeDrawing) { insets ->
                    Column(Modifier.fillMaxSize().padding(insets).padding(if (expanded) 0.dp else 8.dp),
                        verticalArrangement = if (choosingProfile) Arrangement.spacedBy(24.dp, Alignment.CenterVertically) else Arrangement.spacedBy(if (expanded) 0.dp else 4.dp)) {
                        // Center the chooser as one group, keeping generous gaps between profile rows.
                        if (!expanded) Surface(
                            modifier = Modifier.fillMaxWidth(),
                            color = palette.surface, shape = MaterialTheme.shapes.medium) {
                            if (state.profile != null) Row(Modifier.fillMaxWidth().padding(8.dp),
                                verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.SpaceBetween) {
                                PhoneBrand(onHome = model::closeSettings)
                                ProfileMenu(state, model)
                            } else Column(Modifier.fillMaxWidth().padding(8.dp), horizontalAlignment = Alignment.CenterHorizontally) {
                                PhoneBrand(onHome = model::closeSettings)
                                if (choosingProfile) Text("Who’s Watching?", style = MaterialTheme.typography.headlineMedium)
                            }
                        }
                        if (!expanded) {
                            state.error?.let { message ->
                                Text(message, color = MaterialTheme.colorScheme.error)
                                TextButton(onClick = model::dismissError) { Text("Dismiss") }
                            }
                            if (state.busy) LinearProgressIndicator(Modifier.fillMaxWidth())
                        }
                        when {
                            !state.signedIn -> LoginScreen(state, model)
                            state.mustChangePassword -> AccountScreen(state, model, forced = true)
                            state.settings == "account" -> AccountScreen(state, model)
                            state.settings == "profile" -> ProfileSettingsScreen(state, model)
                            state.profile == null -> ProfileScreen(state, model)
                            else -> {
                                if (state.channel != null) {
                                    // Keep the player at the same Compose call site; fullscreen changes bounds, not its identity.
                                    val playerModifier = if (expanded) Modifier.fillMaxWidth().weight(1f)
                                        else Modifier.fillMaxWidth().aspectRatio(16f / 9f)
                                    PhonePlayer(state, model, playerModifier, expanded, pictureInPicture = pip,
                                        onFullscreen = { fullscreen = !fullscreen })
                                }
                                if (!expanded) ChannelGuide(state, model, palette, Modifier.weight(1f))
                            }
                        }
                    }
                }
            }
        }
    }
}

/** Profile administration is separate from the account's Owner authorization. */
@Composable
private fun ProfileMenu(state: PhoneState, model: PhoneModel) {
    var open by remember { mutableStateOf(false) }
    Box {
        TextButton(onClick = { open = true }) { ProfileName(state.profile!!, model) }
        DropdownMenu(expanded = open, onDismissRequest = { open = false }) {
            DropdownMenuItem(text = { Text("Switch Profile") }, onClick = { open = false; model.changeProfile() })
            DropdownMenuItem(text = { Text("Profile Settings") }, onClick = { open = false; model.openSettings("profile") })
            DropdownMenuItem(text = { Text("Sign Out") }, onClick = { open = false; model.logout() })
        }
    }
}

/** Password values remain in memory and disappear when this screen is removed. */
@Composable
private fun AccountScreen(state: PhoneState, model: PhoneModel, forced: Boolean = false) {
    var current by remember { mutableStateOf("") }
    var username by remember(state.username) { mutableStateOf(state.username) }
    var password by remember { mutableStateOf("") }
    var confirmation by remember { mutableStateOf("") }
    LaunchedEffect(state.notice) { if (state.notice != null) { current = ""; password = ""; confirmation = "" } }
    LazyColumn(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        item { Text(if (forced) "Change Password" else "Admin Settings", style = MaterialTheme.typography.headlineSmall) }
        if (forced) item { Text("Change your password before continuing.") }
        if (state.passwordLocked) item { Text("Your account’s username and password are locked. Contact an Owner for changes.") }
        if (!state.passwordLocked) item { AccountPasswordField("Current Password", current, { current = it }) }
        if (!forced) item { OutlinedTextField(username, { username = it }, readOnly = state.passwordLocked,
            label = { Text("Username") }, singleLine = true, modifier = Modifier.fillMaxWidth()) }
        if (!state.passwordLocked) {
            item { AccountPasswordField("New Password", password, { password = it }) }
            item { AccountPasswordField("Confirm New Password", confirmation, { confirmation = it }) }
            item { Button(onClick = { model.saveAccount(current, username, password, confirmation) },
                enabled = !state.busy && current.isNotEmpty() && username.isNotBlank() && (!forced || password.isNotEmpty())) { Text("Save Changes") } }
        }
        state.notice?.let { item { Text(it) } }
        item { TextButton(onClick = if (forced) model::logout else model::closeSettings, enabled = !state.busy) { Text(if (forced) "Sign Out" else "Back") } }
    }
}

@Composable
private fun AccountPasswordField(label: String, value: String, changed: (String) -> Unit) {
    var visible by remember { mutableStateOf(false) }
    OutlinedTextField(value, changed, label = { Text(label) }, singleLine = true,
        visualTransformation = if (visible) VisualTransformation.None else PasswordVisualTransformation(),
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password), modifier = Modifier.fillMaxWidth(),
        trailingIcon = { TextButton(onClick = { visible = !visible }) { Text(if (visible) "Hide" else "Show") } })
}

@Composable
private fun LoginScreen(state: PhoneState, model: PhoneModel) {
    var address by rememberSaveable { mutableStateOf(state.origin) }
    var bypassTls by rememberSaveable(address) { mutableStateOf(state.bypassTls && serverOrigin(address) == state.origin) }
    var username by remember { mutableStateOf("") }
    // Password deliberately excludes rememberSaveable and all persistence.
    var password by remember { mutableStateOf("") }
    var showPassword by remember { mutableStateOf(false) }
    val keyboard = LocalSoftwareKeyboardController.current
    // Request the IME on focus; emulator hardware-keyboard policy is configured separately.
    val fieldModifier = Modifier.fillMaxWidth().onFocusChanged { if (it.isFocused) keyboard?.show() }
    val eye = remember(showPassword) {
        val shape = if (showPassword)
            "M12 4.5C7 4.5 2.7 7.6 1 12c1.7 4.4 6 7.5 11 7.5s9.3-3.1 11-7.5c-1.7-4.4-6-7.5-11-7.5zm0 12.5a5 5 0 1 1 0-10 5 5 0 0 1 0 10zm0-8a3 3 0 1 0 0 6 3 3 0 0 0 0-6z"
        else "M2 2l20 20-1.4 1.4-4.1-4.1c-1.4.5-2.9.7-4.5.7-5 0-9.3-3.1-11-7.5 1-2.4 2.7-4.4 4.8-5.7L.6 3.4 2 2zm5.2 6.2A5 5 0 0 0 15.8 17L7.2 8.2zM12 4.5c5 0 9.3 3.1 11 7.5a13.7 13.7 0 0 1-3.4 4.9l-3-3a5 5 0 0 0-6.5-6.5L7.7 5c1.4-.3 2.8-.5 4.3-.5z"
        ImageVector.Builder(defaultWidth = 24.dp, defaultHeight = 24.dp, viewportWidth = 24f, viewportHeight = 24f)
            .addPath(PathParser().parsePathString(shape).toNodes(), fill = SolidColor(Color.Black)).build()
    }
    LazyColumn(contentPadding = PaddingValues(top = 40.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        item { Text("Connect to your AlreadyOn server") }
        if (state.origin.isNotBlank()) item { TextButton(onClick = model::reconnect, enabled = !state.busy) { Text("Reconnect saved session") } }
        item { OutlinedTextField(address, { address = it }, label = { Text("HTTPS server address") }, singleLine = true, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri), modifier = fieldModifier) }
        item {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Checkbox(checked = bypassTls, onCheckedChange = { bypassTls = it })
                Text("Bypass TLS certificate verification")
            }
            if (bypassTls) Text("This server’s identity will not be verified. Use only for a server you trust on a trusted network.", color = MaterialTheme.colorScheme.error)
        }
        item { OutlinedTextField(username, { username = it }, label = { Text("Username") }, singleLine = true, modifier = fieldModifier) }
        item { OutlinedTextField(password, { password = it }, label = { Text("Password") }, singleLine = true,
            visualTransformation = if (showPassword) VisualTransformation.None else PasswordVisualTransformation(),
            trailingIcon = { IconButton(onClick = { showPassword = !showPassword }) {
                Icon(eye, contentDescription = if (showPassword) "Hide password" else "Show password")
            } }, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password), modifier = fieldModifier) }
        item { Button(onClick = { model.login(address, username, password, bypassTls); password = "" }, enabled = !state.busy && username.isNotBlank() && password.isNotEmpty()) { Text("Sign in") } }
    }
}

@Composable
private fun ProfileScreen(state: PhoneState, model: PhoneModel) {
    var selected by remember { mutableStateOf<ViewerProfile?>(null) }
    var pin by remember { mutableStateOf("") }
    val linkPixels = TypedValue.applyDimension(TypedValue.COMPLEX_UNIT_PT, 2f, LocalContext.current.resources.displayMetrics)
    val linkIncrease = with(LocalDensity.current) { linkPixels.toSp() }
    val linkStyle = MaterialTheme.typography.labelLarge.copy(
        fontSize = (MaterialTheme.typography.labelLarge.fontSize.value + linkIncrease.value).sp)
    // A single scrollable, centered chooser retains room for long profile lists and small screens.
    LazyColumn(Modifier.fillMaxWidth().padding(horizontal = 8.dp),
        horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(24.dp)) {
        items(state.profiles, key = { it.id }) { profile ->
            Button(onClick = { if (profile.hasPin) { selected = profile; pin = "" } else model.selectProfile(profile, "") },
                enabled = !state.busy, modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp), shape = RoundedCornerShape(4.dp)) {
                ProfileName(profile, model)
            }
        }
        if (state.profiles.isEmpty()) item {
            Text("Create a viewer profile in your server’s web interface, then refresh.", textAlign = TextAlign.Center)
        }
        item {
            Row(horizontalArrangement = Arrangement.Center) {
                TextButton(onClick = model::refreshProfiles, enabled = !state.busy) { Text("Refresh", style = linkStyle) }
                TextButton(onClick = model::logout, enabled = !state.busy) { Text("Sign Out", style = linkStyle) }
            }
        }
    }
    selected?.let { profile ->
        AlertDialog(onDismissRequest = { selected = null; pin = "" }, title = { Text("PIN for ${profile.name}") },
            text = { OutlinedTextField(pin, { pin = it }, label = { Text("Profile PIN") }, singleLine = true,
                visualTransformation = PasswordVisualTransformation(), keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword)) },
            confirmButton = { TextButton(onClick = { model.selectProfile(profile, pin); pin = ""; selected = null }, enabled = !state.busy && pin.isNotEmpty()) { Text("Continue") } },
            dismissButton = { TextButton(onClick = { pin = ""; selected = null }) { Text("Cancel") } })
    }
}

@Composable
private fun ChannelGuide(state: PhoneState, model: PhoneModel, palette: PhonePalette, modifier: Modifier) {
    val channels = state.channels
    val format = remember(state.profile?.timeFormat) { DateTimeFormatter.ofPattern(when (state.profile?.timeFormat) {
        "24h" -> "HH:mm"; "12h-padded" -> "hh:mma"; else -> "h:mma"
    }, Locale.US).withZone(ZoneId.systemDefault()) }
    val rows = rememberLazyListState()
    LaunchedEffect(state.channel?.id, channels) {
        val index = channels.indexOfFirst { it.id == state.channel?.id }
        if (index >= 0) rows.scrollToItem(index)
    }
    val scroll = rememberScrollState()
    // Labels and program geometry share a local half-hour anchor, while Now uses the actual clock.
    var now by remember { mutableLongStateOf(System.currentTimeMillis()) }
    LaunchedEffect(Unit) {
        while (true) { now = System.currentTimeMillis(); delay(30_000) }
    }
    val localNow = Instant.ofEpochMilli(now).atZone(ZoneId.systemDefault())
    val origin = localNow.withMinute(localNow.minute / 30 * 30).withSecond(0).withNano(0)
        .toInstant().toEpochMilli()
    val minutesToDp = 4f
    val width = (12 * 60 * minutesToDp).dp
    Column(modifier) {
        Row(Modifier.fillMaxWidth().background(palette.alternate)) {
            Text("CHANNEL", Modifier.width(100.dp).padding(4.dp), color = palette.heading)
            Row(Modifier.horizontalScroll(scroll).width(width)) {
                repeat(24) { column -> Text(format.format(Instant.ofEpochMilli(origin + column * 30 * 60_000L)),
                    Modifier.width((30 * minutesToDp).dp).padding(4.dp), color = palette.muted) }
            }
        }
        LazyColumn(state = rows, verticalArrangement = Arrangement.spacedBy(4.dp)) {
            if (channels.isEmpty()) item { Text("No channels are available for this profile.") }
            items(channels, key = { it.id }) { channel ->
                Row(Modifier.fillMaxWidth().height(72.dp)) {
                    Surface(onClick = { model.watch(channel) }, color = palette.alternate,
                        shape = MaterialTheme.shapes.small, modifier = Modifier.width(100.dp).fillMaxHeight()) {
                        Column(Modifier.padding(4.dp), verticalArrangement = Arrangement.Center) {
                            Text(channel.number.toString(), style = MaterialTheme.typography.titleLarge.copy(fontSize = 18.sp))
                            Text(channel.name, maxLines = 2, style = MaterialTheme.typography.bodySmall)
                        }
                    }
                    Box(Modifier.weight(1f).horizontalScroll(scroll)) {
                        Box(Modifier.width(width).fillMaxHeight()) {
                            channel.programs.forEach { program ->
                                val begin = Instant.parse(program.startsAt).toEpochMilli()
                                val end = Instant.parse(program.endsAt).toEpochMilli()
                                val left = ((begin - origin).coerceAtLeast(0) / 60_000f * minutesToDp).dp
                                val block = ((end.coerceAtMost(origin + 12 * 60 * 60_000L) - begin.coerceAtLeast(origin)) / 60_000f * minutesToDp).coerceAtLeast(0f).dp
                                if (block.value > 0) {
                                    val current = begin <= now && end > now
                                    Surface(onClick = { model.watch(channel) }, modifier = Modifier.offset(x = left).width(block).fillMaxHeight().padding(start = 3.dp)
                                            .then(if (current && channel.id == state.channel?.id) Modifier.border(2.dp, Color.Red, MaterialTheme.shapes.small) else Modifier),
                                        color = if (current) palette.now else palette.surface, shape = MaterialTheme.shapes.small) {
                                        Column(Modifier.padding(4.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                                            Text(if (current) "Now · ${format.format(Instant.ofEpochMilli(end))}" else format.format(Instant.ofEpochMilli(begin)),
                                                color = if (current) palette.nowText else palette.muted, style = MaterialTheme.typography.labelSmall, maxLines = 1)
                                            Text(program.title, color = if (current) palette.nowText else palette.text, maxLines = 2, overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis, style = MaterialTheme.typography.bodyMedium)
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

/** Accept only HTTPS origins so API requests cannot inherit credentials or arbitrary paths. */
internal fun serverOrigin(input: String): String? {
    val uri = runCatching { URI(input.trim()) }.getOrNull() ?: return null
    if (!uri.scheme.equals("https", ignoreCase = true) || uri.host.isNullOrBlank() ||
        uri.rawUserInfo != null || uri.rawQuery != null || uri.rawFragment != null ||
        (uri.rawPath != "" && uri.rawPath != "/") || (uri.port != -1 && uri.port !in 1..65535)) return null
    return URI("https", null, uri.host.lowercase(Locale.ROOT), uri.port, null, null, null).toString().toHttpUrl().toString().removeSuffix("/")
}

// Physical-phone loopback identifies the phone itself, not the selected network server.
internal fun isLoopback(host: String): Boolean {
    val normalized = host.lowercase(Locale.ROOT).removeSuffix(".").removePrefix("[").removeSuffix("]")
    return normalized == "localhost" || normalized.endsWith(".localhost") || normalized == "::1" ||
        normalized == "0:0:0:0:0:0:0:1" || normalized == "0.0.0.0" || normalized.startsWith("127.")
}

// Keep the app wordmark and launcher icon on the shared, font-free brand paths.
@Composable
private fun PhoneBrand(onHome: () -> Unit) {
    // Wordmark visible height is 88/601 of its width; icon padding is 80/108.
    val wordmarkHeight = 170.dp * (88f / 601f)
    val iconSize = wordmarkHeight * 1.05f * (108f / 80f)
    Column(Modifier.clickable(onClickLabel = "AlreadyOn home", onClick = onHome)) {
        Row(verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Image(painterResource(R.drawable.ic_brand), contentDescription = null,
                modifier = Modifier.size(iconSize).offset(y = -iconSize * (5.5f / 108f) - wordmarkHeight * .025f),
                contentScale = ContentScale.Fit)
            Image(painterResource(R.drawable.brand_wordmark), contentDescription = "AlreadyOn",
                modifier = Modifier.width(170.dp).height(32.dp), contentScale = ContentScale.Fit)
        }
        Text("Your library, tuned to you", modifier = Modifier.padding(start = iconSize + 8.dp).offset(y = (-4).dp),
            style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}
