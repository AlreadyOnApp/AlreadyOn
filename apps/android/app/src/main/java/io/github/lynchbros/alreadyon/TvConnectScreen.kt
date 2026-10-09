package io.github.lynchbros.alreadyon

import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import okhttp3.HttpUrl.Companion.toHttpUrl
import okhttp3.OkHttpClient
import okhttp3.Request
import org.json.JSONObject
import java.io.IOException
import java.io.ByteArrayInputStream
import java.security.MessageDigest
import java.security.cert.CertificateFactory
import java.util.concurrent.TimeUnit

private data class TvTrust(val certificate: String, val fingerprint: String)

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
    val certificate = CertificateFactory.getInstance("X.509").generateCertificate(ByteArrayInputStream(pem.toByteArray()))
    val fingerprint = MessageDigest.getInstance("SHA-256").digest(certificate.encoded).joinToString("") { "%02x".format(it) }
    // Compare actual certificate bytes, not an unauthenticated advertised digest.
    return@withContext TvTrust(pem, fingerprint)
}

@Composable
internal fun TvConnectScreen(state: PhoneState, model: PhoneModel, modifier: Modifier) {
    val context = LocalContext.current
    val servers = remember { mutableStateListOf<TvServer>() }
    var scan by remember { mutableIntStateOf(0) }
    var selected by remember { mutableStateOf<TvServer?>(null) }
    var trust by remember { mutableStateOf<TvTrust?>(null) }
    var confirmed by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var checking by remember { mutableStateOf(false) }
    var pin by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    DisposableEffect(scan) {
        val discovery = TvDiscovery(context, found = { server ->
            servers.removeAll { it.name == server.name }; servers.add(server)
        }, lost = { name -> servers.removeAll { it.name == name } }, failed = { error = it })
        onDispose { discovery.close() }
    }
    Column(modifier.fillMaxWidth().verticalScroll(rememberScrollState()).padding(12.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        if (selected == null) {
            Text("Connect to your AlreadyOn server", style = MaterialTheme.typography.headlineSmall)
            Text("Select a server on your home network.")
            if (servers.isEmpty()) Text("Searching… Keep the TV and server on the same network. Guest networks may block discovery.")
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                servers.forEach { server ->
                    Button(modifier = Modifier.tvFocusOutline(), onClick = {
                        selected = server; checking = true; error = null
                        scope.launch {
                            try { trust = probe(server) }
                            catch (e: kotlinx.coroutines.CancellationException) { throw e }
                            catch (e: Exception) { error = e.message ?: "Could not check server." }
                            finally { checking = false }
                        }
                    }, enabled = !checking && !state.busy) { Text("${server.name} · ${server.origin.removePrefix("https://")}") }
                }
            }
            Button(modifier = Modifier.tvFocusOutline(), onClick = { servers.clear(); error = null; scan++ }) { Text("Rescan") }
        } else {
            Text("${selected!!.name} · ${selected!!.origin.removePrefix("https://")}", style = MaterialTheme.typography.headlineSmall)
            if (checking) Text("Checking server…")
            if (trust != null && !confirmed) {
                Text("Compare this certificate fingerprint with TV Sign-In in the server’s Admin Settings.")
                Text(trust!!.fingerprint)
                Button(modifier = Modifier.tvFocusOutline(), onClick = { confirmed = true }) { Text("Confirm Server") }
            }
            if (confirmed) {
                Text("Enter your 6–12 digit TV PIN")
                Text("●".repeat(pin.length), style = MaterialTheme.typography.headlineMedium)
                listOf(listOf("1", "2", "3"), listOf("4", "5", "6"), listOf("7", "8", "9"), listOf("Clear", "0", "Delete")).forEach { row ->
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        row.forEach { digit -> Button(onClick = {
                            pin = when (digit) { "Clear" -> ""; "Delete" -> pin.dropLast(1); else -> (pin + digit).take(12) }
                        }, enabled = !state.busy, modifier = Modifier.width(100.dp).tvFocusOutline()) { Text(digit) } }
                    }
                }
                Button(modifier = Modifier.tvFocusOutline(), onClick = { model.pairTV(selected!!.origin, pin, trust!!.certificate); pin = "" },
                    enabled = pin.length in 6..12 && !state.busy) { Text("Connect") }
            }
            Button(modifier = Modifier.tvFocusOutline(), onClick = { selected = null; trust = null; confirmed = false; pin = ""; error = null; model.dismissError() }, enabled = !checking && !state.busy) { Text("Choose Another Server") }
        }
        error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
    }
}
