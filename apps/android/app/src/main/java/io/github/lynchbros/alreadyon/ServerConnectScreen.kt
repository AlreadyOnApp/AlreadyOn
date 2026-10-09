// Shared phone/tablet and TV discovery, address fallback and account PIN onboarding.
package io.github.lynchbros.alreadyon

import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
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
    var remaining by remember { mutableIntStateOf(30) }
    val scope = rememberCoroutineScope()
    LaunchedEffect(scan) {
        remaining = 30
        while (remaining > 0) { delay(1000); remaining-- }
    }
    fun select(server: TvServer) {
        selected = server; checking = true; error = null; trust = null
        scope.launch {
            try { trust = probe(server) }
            catch (e: kotlinx.coroutines.CancellationException) { throw e }
            catch (e: Exception) { error = e.message ?: "Could not check server." }
            finally { checking = false }
        }
    }
    DisposableEffect(scan, remaining == 0) {
        if (remaining == 0) return@DisposableEffect onDispose {}
        val discovery = TvDiscovery(context, found = { server ->
            servers.removeAll { it.name == server.name }; servers.add(server)
        }, lost = { name -> servers.removeAll { it.name == name } }, failed = { error = it })
        onDispose { discovery.close() }
    }
    Column(modifier.fillMaxWidth().verticalScroll(rememberScrollState()).padding(12.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        if (selected == null) {
            Text("Connect to your AlreadyOn server", style = MaterialTheme.typography.headlineSmall)
            Text("Select a server on your home network.")
            if (remaining > 0) Text("Searching for local servers… ${remaining}s remaining")
            else Text(if (servers.isEmpty()) "No local server found. Enter its HTTPS address below." else "Select a server or enter an HTTPS address below.")
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                servers.forEach { server ->
                    Button(modifier = Modifier.tvFocusOutline(), onClick = {
                        select(server)
                    }, enabled = !checking && !state.busy) { Text("${server.name} · ${server.origin.removePrefix("https://")}") }
                }
            }
            if (remaining == 0) {
                OutlinedTextField(address, { address = it }, label = { Text("HTTPS server address") },
                    singleLine = true, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri), modifier = Modifier.fillMaxWidth())
                Button(modifier = Modifier.tvFocusOutline(), onClick = {
                    val origin = serverOrigin(address)
                    if (origin == null) error = "Enter a valid HTTPS server address."
                    else select(TvServer("AlreadyOn server", origin))
                }, enabled = address.isNotBlank() && !checking && !state.busy) { Text("Check Server") }
            }
            Button(modifier = Modifier.tvFocusOutline(), onClick = { servers.clear(); error = null; remaining = 30; scan++ }) { Text("Rescan") }
        } else {
            Text("${selected!!.name} · ${selected!!.origin.removePrefix("https://")}", style = MaterialTheme.typography.headlineSmall)
            if (checking) Text("Checking server…")
            if (trust != null) {
                OutlinedTextField(username, { username = it }, label = { Text("Username") },
                    singleLine = true, modifier = Modifier.fillMaxWidth(), enabled = !state.busy)
                Text("Enter your 6–12 digit TV PIN")
                if (!BuildConfig.IS_TV) OutlinedTextField(pin, { pin = it.filter { digit -> digit in '0'..'9' }.take(12) },
                    label = { Text("TV PIN") }, singleLine = true, visualTransformation = PasswordVisualTransformation(),
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword), modifier = Modifier.fillMaxWidth(), enabled = !state.busy)
                else {
                    Text("●".repeat(pin.length), style = MaterialTheme.typography.headlineMedium)
                    listOf(listOf("1", "2", "3"), listOf("4", "5", "6"), listOf("7", "8", "9"), listOf("Clear", "0", "Delete")).forEach { row ->
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            row.forEach { digit -> Button(onClick = {
                                pin = when (digit) { "Clear" -> ""; "Delete" -> pin.dropLast(1); else -> (pin + digit).take(12) }
                            }, enabled = !state.busy, modifier = Modifier.width(100.dp).tvFocusOutline()) { Text(digit) } }
                        }
                    }
                }
                Button(modifier = Modifier.tvFocusOutline(), onClick = { model.signInWithPIN(selected!!.origin, username, pin, trust!!.certificate); pin = "" },
                    enabled = username.isNotBlank() && pin.length in 6..12 && !state.busy) { Text("Connect") }
            }
            Button(modifier = Modifier.tvFocusOutline(), onClick = { selected = null; trust = null; pin = ""; error = null; model.dismissError() }, enabled = !checking && !state.busy) { Text("Choose Another Server") }
        }
        error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
    }
}
