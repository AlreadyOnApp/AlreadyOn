// Shared phone/tablet and TV discovery, address fallback and account PIN onboarding.
package io.github.lynchbros.alreadyon

import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.ui.Alignment
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.input.key.*
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import kotlinx.coroutines.delay
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import okhttp3.HttpUrl.Companion.toHttpUrl
import okhttp3.OkHttpClient
import okhttp3.Request
import org.json.JSONObject
import java.io.IOException
import java.io.ByteArrayInputStream
import java.security.cert.CertificateFactory
import java.util.concurrent.TimeUnit

private data class TvTrust(val certificate: String)

/** Only public bootstrap metadata uses an untrusted TLS connection; PINs never do. */
private suspend fun probe(server: TvServer): TvTrust = withContext(Dispatchers.IO) {
    val base = server.origin.toHttpUrl()
    val http = OkHttpClient.Builder().serverTls(base, true).connectTimeout(5, TimeUnit.SECONDS)
        .callTimeout(10, TimeUnit.SECONDS).build()
    fun get(path: String): JSONObject = http.newCall(Request.Builder().url(base.resolve(path)!!).build()).execute().use { response ->
        if (!response.isSuccessful) throw IOException("Server is not available for local TV pairing.")
        val source = response.body?.source() ?: throw IOException("Empty server response.")
        if (source.request(64 * 1024L + 1)) throw IOException("Server metadata is too large.")
        JSONObject(source.readUtf8())
    }
    val info = get("/api/tv/server")
    if (info.optInt("protocol") != 1 || !info.optBoolean("pairingEnabled")) throw IOException("Set your TV PIN in this server’s Admin Settings first.")
    val data = get("/api/tv/trust")
    val pem = data.getString("certificate")
    CertificateFactory.getInstance("X.509").generateCertificate(ByteArrayInputStream(pem.toByteArray()))
    return@withContext TvTrust(pem)

}

@Composable
internal fun ServerConnectScreen(state: PhoneState, model: PhoneModel, modifier: Modifier) {
    val context = LocalContext.current
    val servers = remember { mutableStateListOf<TvServer>() }
    var scan by remember { mutableIntStateOf(0) }
    var selected by remember { mutableStateOf<TvServer?>(null) }
    var trust by remember { mutableStateOf<TvTrust?>(null) }
    var error by remember { mutableStateOf<String?>(null) }
    var checking by remember { mutableStateOf(false) }
    var pin by remember { mutableStateOf("") }
    var username by remember { mutableStateOf("") }
    var address by remember { mutableStateOf(state.origin) }
    var manualEntry by remember { mutableStateOf(false) }
    var remaining by remember { mutableIntStateOf(30) }
    val scope = rememberCoroutineScope()
    val keyboard = LocalSoftwareKeyboardController.current
    val focusManager = LocalFocusManager.current
    val pinFocus = remember { FocusRequester() }
    val connectFocus = remember { FocusRequester() }
    var pinKeypad by remember { mutableStateOf(false) }
    fun enterPin() { keyboard?.hide(); pinFocus.requestFocus(); pinKeypad = true }
    fun closePin() {
        pinKeypad = false
        focusManager.clearFocus(force = true)
        if (username.isNotBlank() && pin.length in 6..12) connectFocus.requestFocus()
    }
    LaunchedEffect(scan, manualEntry) {
        if (manualEntry) return@LaunchedEffect
        remaining = 30
        while (remaining > 0) { delay(1000); remaining-- }
    }
    fun select(server: TvServer) {
        selected = server; checking = true; error = null; trust = null; model.dismissError()
        scope.launch {
            try { trust = probe(server) }
            catch (e: kotlinx.coroutines.CancellationException) { throw e }
            catch (e: Exception) { error = e.message ?: "Could not check server." }
            finally { checking = false }
        }
    }
    DisposableEffect(scan, remaining == 0, manualEntry, selected) {
        if (remaining == 0 || manualEntry || selected != null) return@DisposableEffect onDispose {}
        val discovery = TvDiscovery(context, found = { server ->
            servers.removeAll { it.name == server.name }; servers.add(server)
        }, lost = { name -> servers.removeAll { it.name == name } }, failed = { error = it })
        onDispose { discovery.close() }
    }
    Box(modifier.fillMaxWidth(), contentAlignment = if (BuildConfig.IS_TV) Alignment.Center else Alignment.TopStart) {
    Column(Modifier.widthIn(max = if (BuildConfig.IS_TV) 560.dp else androidx.compose.ui.unit.Dp.Infinity).fillMaxWidth()
        .verticalScroll(rememberScrollState()).padding(12.dp),
        horizontalAlignment = if (BuildConfig.IS_TV) Alignment.CenterHorizontally else Alignment.Start,
        verticalArrangement = Arrangement.spacedBy(12.dp)) {
        if (selected == null) {
            Text("Connect to your AlreadyOn server", style = MaterialTheme.typography.headlineSmall)
            Text("Select a server on your home network.")
            if (manualEntry) Text("Enter your server’s HTTPS address.")
            else if (remaining > 0) Text("Searching for local servers… ${remaining}s remaining")
            else Text(if (servers.isEmpty()) "No local server found. Enter its HTTPS address below." else "Select a server or enter an HTTPS address below.")
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                servers.forEach { server ->
                    Button(modifier = Modifier.tvFocusOutline(), onClick = {
                        select(server)
                    }, enabled = !checking && !state.busy) { Text("${server.name} · ${server.origin.removePrefix("https://")}") }
                }
            }
            if (BuildConfig.IS_TV || remaining == 0) {
                OutlinedTextField(address, { address = it; manualEntry = true }, label = { Text("HTTPS server address") },
                    singleLine = true, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri), modifier = Modifier.fillMaxWidth())
                Button(modifier = Modifier.tvFocusOutline(), onClick = {
                    val origin = serverOrigin(address)
                    if (origin == null) error = "Enter a valid HTTPS server address."
                    else select(TvServer("AlreadyOn server", origin))
                }, enabled = address.isNotBlank() && !checking && !state.busy) { Text("Check Server") }
            }
            Button(modifier = Modifier.tvFocusOutline(), onClick = { servers.clear(); error = null; manualEntry = false; remaining = 30; scan++ }) { Text("Rescan") }
        } else {
            Text("${selected!!.name} · ${selected!!.origin.removePrefix("https://")}", style = MaterialTheme.typography.headlineSmall)
            if (checking) Text("Checking server…")
            if (trust != null) {
                OutlinedTextField(username, { username = it }, label = { Text("Username") },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(imeAction = if (BuildConfig.IS_TV) ImeAction.Next else ImeAction.Default),
                    keyboardActions = KeyboardActions(onNext = { if (BuildConfig.IS_TV) enterPin() }, onDone = { if (BuildConfig.IS_TV) enterPin() }),
                    modifier = Modifier.fillMaxWidth(if (BuildConfig.IS_TV) 0.6f else 1f).onPreviewKeyEvent {
                        if (BuildConfig.IS_TV && (it.key == Key.Enter || it.key == Key.NumPadEnter)) {
                            if (it.type == KeyEventType.KeyUp) enterPin()
                            true
                        } else false
                    }, enabled = !state.busy)
                Text("Enter your 6–12 digit TV PIN")
                if (!BuildConfig.IS_TV) OutlinedTextField(pin, { pin = it.filter { digit -> digit in '0'..'9' }.take(12) },
                    label = { Text("TV PIN") }, singleLine = true, visualTransformation = PasswordVisualTransformation(),
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword), modifier = Modifier.fillMaxWidth(), enabled = !state.busy)
                else OutlinedTextField(pin, {}, label = { Text("TV PIN") }, readOnly = true,
                    singleLine = true, visualTransformation = PasswordVisualTransformation(),
                    modifier = Modifier.fillMaxWidth(0.6f).focusRequester(pinFocus).onFocusChanged {
                        if (it.isFocused) { keyboard?.hide(); pinKeypad = true }
                    }, enabled = !state.busy)
                Button(modifier = Modifier.focusRequester(connectFocus).tvFocusOutline(), onClick = { model.signInWithPIN(selected!!.origin, username, pin, trust!!.certificate); pin = "" },
                    enabled = username.isNotBlank() && pin.length in 6..12 && !state.busy) { Text("Connect") }
            }
            Button(modifier = Modifier.tvFocusOutline(), onClick = { selected = null; trust = null; pin = ""; error = null; model.dismissError() }, enabled = !checking && !state.busy) { Text("Choose Another Server") }
        }
        error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
        if (BuildConfig.IS_TV) state.error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
    }
    }
    if (BuildConfig.IS_TV && pinKeypad) Dialog(onDismissRequest = { closePin() }) {
        val firstDigit = remember { FocusRequester() }
        val keypadView = LocalView.current
        // The dialog must own window focus before assigning focus to its first key.
        LaunchedEffect(Unit) {
            while (!keypadView.hasWindowFocus()) delay(16)
            firstDigit.requestFocus()
        }
        Surface(shape = MaterialTheme.shapes.large) {
            Column(Modifier.width(360.dp).padding(20.dp), horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text("TV PIN", style = MaterialTheme.typography.headlineSmall)
                Text("●".repeat(pin.length), style = MaterialTheme.typography.headlineMedium)
                listOf(listOf("1", "2", "3"), listOf("4", "5", "6"), listOf("7", "8", "9"), listOf("Clear", "0", "Delete")).forEach { row ->
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        row.forEach { digit ->
                            Button(onClick = {
                                pin = when (digit) { "Clear" -> ""; "Delete" -> pin.dropLast(1); else -> (pin + digit).take(12) }
                            }, modifier = Modifier.width(100.dp).then(if (digit == "1") Modifier.focusRequester(firstDigit) else Modifier).tvFocusOutline()) { Text(digit) }
                        }
                    }
                }
                Button(onClick = { closePin() }, modifier = Modifier.tvFocusOutline()) { Text("✓ Done") }
            }
        }
    }
}
