package io.github.lynchbros.alreadyon

import android.content.Context
import android.net.nsd.NsdManager
import android.net.nsd.NsdServiceInfo
import android.net.wifi.WifiManager
import kotlinx.coroutines.*
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.suspendCancellableCoroutine
import java.net.InetAddress
import java.net.URI
import kotlin.coroutines.resume

internal data class TvServer(val name: String, val origin: String)

/** DNS-SD discovers endpoints, never usernames, PINs or automatic trust grants. */
internal class TvDiscovery(context: Context, private val found: (TvServer) -> Unit,
    private val lost: (String) -> Unit, private val failed: (String) -> Unit) {
    private val manager = context.getSystemService(NsdManager::class.java)
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
    private val pending = Channel<NsdServiceInfo>(32)
    private val wifi = context.applicationContext.getSystemService(WifiManager::class.java)
    private val lock = wifi?.createMulticastLock("alreadyon-tv-discovery")?.apply { setReferenceCounted(false) }
    private var started = false
    private val listener = object : NsdManager.DiscoveryListener {
        override fun onDiscoveryStarted(type: String) = Unit
        override fun onDiscoveryStopped(type: String) = Unit
        override fun onStartDiscoveryFailed(type: String, code: Int) { scope.launch { failed("Local discovery could not start. Check the network and rescan.") } }
        override fun onStopDiscoveryFailed(type: String, code: Int) = Unit
        override fun onServiceFound(service: NsdServiceInfo) { pending.trySend(service) }
        override fun onServiceLost(service: NsdServiceInfo) { scope.launch { lost(service.serviceName) } }
    }
    init {
        // Legacy NSD resolution allows one outstanding resolve; serialize its queue.
        scope.launch {
            for (service in pending) {
                val resolved = resolve(service) ?: continue
                val address = resolved.host ?: continue
                if (!privateAddress(address) || resolved.port !in 1..65535) continue
                val origin = URI("https", null, address.hostAddress, resolved.port, null, null, null).toString()
                found(TvServer(resolved.serviceName.take(128), origin))
            }
        }
        try {
            lock?.acquire()
            manager.discoverServices("_alreadyon._tcp.", NsdManager.PROTOCOL_DNS_SD, listener)
            started = true
        } catch (_: Exception) { failed("Local discovery is unavailable. Check the network and rescan.") }
    }
    @Suppress("DEPRECATION")
    private suspend fun resolve(service: NsdServiceInfo): NsdServiceInfo? = suspendCancellableCoroutine { continuation ->
        try {
            manager.resolveService(service, object : NsdManager.ResolveListener {
                override fun onResolveFailed(info: NsdServiceInfo, code: Int) { if (continuation.isActive) continuation.resume(null) }
                override fun onServiceResolved(info: NsdServiceInfo) { if (continuation.isActive) continuation.resume(info) }
            })
        } catch (_: Exception) { if (continuation.isActive) continuation.resume(null) }
    }
    fun close() {
        if (started) runCatching { manager.stopServiceDiscovery(listener) }
        scope.cancel(); pending.close()
        if (lock?.isHeld == true) lock.release()
    }
}

internal fun privateAddress(address: InetAddress): Boolean = address.isSiteLocalAddress || address.isLoopbackAddress ||
    address.isLinkLocalAddress || (address.address.size == 16 && (address.address[0].toInt() and 0xfe) == 0xfc)
