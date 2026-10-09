package io.github.lynchbros.alreadyon

import android.os.SystemClock
import android.net.nsd.NsdManager
import android.net.nsd.NsdServiceInfo
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import androidx.test.uiautomator.UiDevice
import org.junit.Assert.*
import androidx.lifecycle.ViewModelProvider
import org.junit.Rule
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.*
import org.junit.Test
import org.junit.runner.RunWith
import java.net.ServerSocket
import java.net.Socket
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import kotlin.concurrent.thread

/** Real TV discovery/login against a marked, disposable host server and synthetic media.
 * The TCP bridge places that host fixture on the emulator's own virtual LAN; no
 * production discovery hooks, saved credentials or live databases are involved.
 */
@RunWith(AndroidJUnit4::class)
class TvNavigationTest {
    @get:Rule val compose = createAndroidComposeRule<MainActivity>()
    @Test fun pairingGuideAndRemoteControls() {
        val instrumentation = InstrumentationRegistry.getInstrumentation()
        val port = InstrumentationRegistry.getArguments().getString("serverPort")?.toInt()
            ?: error("Supply serverPort for an isolated synthetic server; see docs/GOOGLE_TV.md")
        val device = UiDevice.getInstance(instrumentation)
        val bridge = ServerSocket(0)
        var running = true
        val peers = java.util.Collections.synchronizedList(mutableListOf<Socket>())
        thread(isDaemon = true) {
            while (running) {
                val incoming = runCatching { bridge.accept() }.getOrNull() ?: break
                thread(isDaemon = true) {
                    val outgoing = Socket("10.0.2.2", port)
                    peers.add(incoming); peers.add(outgoing)
                    thread(isDaemon = true) { runCatching { incoming.getInputStream().copyTo(outgoing.getOutputStream()) }; runCatching { outgoing.shutdownOutput() } }
                    runCatching { outgoing.getInputStream().copyTo(incoming.getOutputStream()) }
                    runCatching { incoming.close() }; runCatching { outgoing.close() }
                }
            }
        }
        val nsd = instrumentation.targetContext.getSystemService(NsdManager::class.java)
        val registered = CountDownLatch(1)
        val listener = object : NsdManager.RegistrationListener {
            override fun onServiceRegistered(info: NsdServiceInfo) { registered.countDown() }
            override fun onRegistrationFailed(info: NsdServiceInfo, code: Int) = Unit
            override fun onServiceUnregistered(info: NsdServiceInfo) = Unit
            override fun onUnregistrationFailed(info: NsdServiceInfo, code: Int) = Unit
        }
        val manual = InstrumentationRegistry.getArguments().getString("entryMode") == "manual"
        val startedAt = SystemClock.elapsedRealtime()
        try {
            if (!manual) nsd.registerService(NsdServiceInfo().apply {
                serviceName = "AlreadyOn TV fixture"; serviceType = "_alreadyon._tcp."
                setPort(bridge.localPort)
            }, NsdManager.PROTOCOL_DNS_SD, listener)
            if (!manual) assertTrue("Fixture discovery registration", registered.await(10, TimeUnit.SECONDS))
            fun exists(text: String) = compose.onAllNodesWithText(text).fetchSemanticsNodes().isNotEmpty()
            fun click(text: String) {
                compose.waitUntil(60000) { exists(text) }
                compose.onAllNodesWithText(text).onFirst().performScrollTo().performClick()
            }
            if (manual) {
                assertFalse("No manual address before search timeout", exists("HTTPS server address"))
                compose.waitUntil(40000) { exists("HTTPS server address") }
                assertTrue("Search lasts 30 seconds", SystemClock.elapsedRealtime() - startedAt >= 28500)
                compose.onNode(hasSetTextAction() and hasText("HTTPS server address")).performTextInput("https://127.0.0.1:${bridge.localPort}")
                device.pressBack() // Dismiss the TV keyboard.
                click("Check Server")
            } else {
                compose.waitUntil(60000) { compose.onAllNodes(hasText("AlreadyOn TV fixture ·", substring = true)).fetchSemanticsNodes().isNotEmpty() }
                compose.onAllNodes(hasText("AlreadyOn TV fixture ·", substring = true)).onFirst().performClick()
            }
            compose.waitUntil(60000) { exists("Username") }
            assertFalse("Server fingerprint confirmation is skipped", exists("Confirm Server"))
            compose.onNode(hasSetTextAction() and hasText("Username")).performTextInput("tv-fixture")
            device.pressBack() // Dismiss the keyboard before using the PIN keypad.
            // Synthetic account PIN set by the isolated test fixture, never a live account.
            "123456".forEach { click(it.toString()) }
            click("Connect")
            click("Admin")
            assertTrue("Compact guide on first entry", exists("CHANNEL"))
            assertTrue("Guide channel 110", exists("110"))
            val model = ViewModelProvider(compose.activity)[PhoneModel::class.java]
            compose.runOnIdle { model.setQuality(720); model.setSubtitle(null) }
            val muted = model.state.value.muted
            click("110")
            assertTrue("Fullscreen hides guide", !exists("CHANNEL"))
            compose.waitUntil(60000) { !exists("Preparing channel…") }
            // Reveal the hidden transport controls using only the remote.
            device.pressDPadDown()
            compose.waitUntil(10000) { compose.onAllNodes(hasContentDescription("Mute") or hasContentDescription("Unmute")).fetchSemanticsNodes().isNotEmpty() }
            device.pressDPadRight()
            compose.waitUntil(60000) { exists("111") }
            device.pressDPadLeft()
            compose.waitUntil(60000) { exists("110") }
            compose.runOnIdle {
                assertEquals("Quality survives tuning", 720, model.state.value.quality)
                assertEquals("Mute survives tuning", muted, model.state.value.muted)
                assertNull("Subtitles remain off", model.state.value.subtitleIndex)
            }
            device.pressBack()
            compose.waitUntil(10000) { exists("CHANNEL") }
            compose.waitForIdle()
            Thread.sleep(1000) // Allow the SurfaceView compositor to present the compact frame.
            device.takeScreenshot(java.io.File(instrumentation.targetContext.getExternalFilesDir(null), "tv-guide.png"))
        } finally {
            device.takeScreenshot(java.io.File(instrumentation.targetContext.getExternalFilesDir(null), "tv-last-state.png"))
            running = false; bridge.close()
            peers.toList().forEach { runCatching { it.close() } }
            runCatching { nsd.unregisterService(listener) }
        }
    }
}
