package io.github.lynchbros.alreadyon

import android.os.SystemClock
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Surface
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.input.pointer.pointerInput
import kotlinx.coroutines.delay

/** Shared static startup cover; the session, guide and player load underneath without another request. */
@Composable
internal fun StartupPreload(content: @Composable () -> Unit) {
    var loading by rememberSaveable { mutableStateOf(true) }
    val deadline = rememberSaveable { SystemClock.elapsedRealtime() + 2_000 }
    // Configuration changes retain the acknowledgement rather than restarting the delay.
    LaunchedEffect(Unit) { delay((deadline - SystemClock.elapsedRealtime()).coerceAtLeast(0)); loading = false }
    Box(Modifier.fillMaxSize().onPreviewKeyEvent { loading }) {
        content()
        if (loading) PhoneTheme {
            Surface(Modifier.fillMaxSize().pointerInput(Unit) {
                awaitPointerEventScope {
                    while (true) awaitPointerEvent().changes.forEach { it.consume() }
                }
            }, color = phonePalette.background) {
                Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { PhoneBrand() }
            }
        }
    }
}
