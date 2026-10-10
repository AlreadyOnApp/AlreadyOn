package io.github.lynchbros.alreadyon

import android.graphics.Paint
import android.text.TextPaint
import android.text.TextUtils
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.*
import androidx.compose.ui.graphics.*
import androidx.compose.ui.graphics.drawscope.drawIntoCanvas
import androidx.compose.ui.input.key.*
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale

/** One focus target per channel avoids hundreds of invisible Compose focus/layout nodes on small TVs. */
@Composable
internal fun TvGuide(state: PhoneState, palette: PhonePalette, modifier: Modifier, resetSignal: Int = 0, onChannelFocus: (Long, Boolean) -> Unit = { _, _ -> }, onTune: (GuideChannel) -> Unit) {
    val channels = state.channels
    val rows = rememberLazyListState()
    val scope = rememberCoroutineScope()
    val targets = remember(channels.map { it.id }) { channels.associate { it.id to FocusRequester() } }
    var now by remember { mutableLongStateOf(System.currentTimeMillis()) }
    var selectedTime by remember { mutableLongStateOf(now) }
    var origin by remember { mutableLongStateOf(now / 1_800_000 * 1_800_000) }
    val format = remember(state.profile?.timeFormat) {
        DateTimeFormatter.ofPattern(when (state.profile?.timeFormat) { "24h" -> "HH:mm"; "12h-padded" -> "hh:mma"; else -> "h:mma" }, Locale.US).withZone(ZoneId.systemDefault())
    }
    LaunchedEffect(Unit) { while (true) { delay(30_000); now = System.currentTimeMillis() } }
    // Restoring the guide always restores the selected channel, independent of stream readiness.
    LaunchedEffect(channels.isNotEmpty(), state.profile?.id, resetSignal) {
        if (channels.isNotEmpty()) {
            val index = channels.indexOfFirst { it.id == state.channel?.id }.coerceAtLeast(0)
            selectedTime = System.currentTimeMillis(); origin = selectedTime / 1_800_000 * 1_800_000
            rows.scrollToItem(index)
            withFrameNanos { }
            targets[channels[index].id]?.requestFocus()
        }
    }
    Column(modifier) {
        Row(Modifier.fillMaxWidth().background(palette.alternate)) {
            Text("CHANNEL", Modifier.width(100.dp).padding(4.dp), color = palette.heading)
            Canvas(Modifier.weight(1f).height(34.dp)) {
                val paint = Paint(Paint.ANTI_ALIAS_FLAG).apply { color = palette.muted.toArgb(); textSize = 14.dp.toPx() }
                var x = 0f
                var stamp = origin
                while (x < size.width) {
                    drawIntoCanvas { it.nativeCanvas.drawText(format.format(Instant.ofEpochMilli(stamp)), x + 4.dp.toPx(), 23.dp.toPx(), paint) }
                    x += 120.dp.toPx(); stamp += 1_800_000
                }
            }
        }
        LazyColumn(state = rows, verticalArrangement = Arrangement.spacedBy(4.dp)) {
            if (channels.isEmpty()) item { Text("No channels are available for this profile.") }
            itemsIndexed(channels, key = { _, channel -> channel.id }) { index, channel ->
                var focused by remember { mutableStateOf(false) }
                val programs = remember(channel.programs) { channel.programs.map {
                    Triple(it, Instant.parse(it.startsAt).toEpochMilli(), Instant.parse(it.endsAt).toEpochMilli())
                } }
                BoxWithConstraints(Modifier.fillMaxWidth()) {
                    val windowMs = ((maxWidth.value - 100) / 4 * 60_000).toLong()
                    fun reportFocus() {
                        val current = programs.find { System.currentTimeMillis() >= it.second && System.currentTimeMillis() < it.third }
                        onChannelFocus(channel.id, current != null && selectedTime >= current.second && selectedTime < current.third)
                    }
                    fun reveal(start: Long) {
                        if (start < origin) origin = start / 1_800_000 * 1_800_000
                        else if (start >= origin + windowMs) origin = start / 1_800_000 * 1_800_000
                    }
                    Box(Modifier.fillMaxWidth().height(72.dp).focusRequester(targets.getValue(channel.id))
                            .onFocusChanged { focused = it.hasFocus; if (it.isFocused) reportFocus() }
                            .semantics(mergeDescendants = true) { contentDescription = "Channel ${channel.number}: ${channel.name}" }
                            .onPreviewKeyEvent { event ->
                                if (event.key !in listOf(Key.DirectionUp, Key.DirectionDown, Key.DirectionLeft, Key.DirectionRight)) false
                                else {
                                    if (event.type == KeyEventType.KeyDown) {
                                        if (event.key == Key.DirectionUp || event.key == Key.DirectionDown) {
                                            val next = index + if (event.key == Key.DirectionDown) 1 else -1
                                            if (next !in channels.indices) return@onPreviewKeyEvent false
                                            scope.launch {
                                                if (rows.layoutInfo.visibleItemsInfo.none { it.index == next && it.offset + it.size <= rows.layoutInfo.viewportEndOffset }) rows.scrollToItem(next)
                                                withFrameNanos { }
                                                targets[channels[next].id]?.requestFocus()
                                            }
                                        } else {
                                            val current = programs.indexOfFirst { selectedTime >= it.second && selectedTime < it.third }.coerceAtLeast(0)
                                            val next = current + if (event.key == Key.DirectionRight) 1 else -1
                                            programs.getOrNull(next)?.let { selectedTime = it.second + 1; reveal(it.second); reportFocus() }
                                        }
                                    }
                                    true
                                }
                            }.clickable(interactionSource = remember { MutableInteractionSource() }, indication = null) { onTune(channel) }) {
                        Row {
                            Column(Modifier.width(100.dp).fillMaxHeight().background(palette.alternate).padding(4.dp), verticalArrangement = Arrangement.Center) {
                                Text(channel.number.toString(), style = MaterialTheme.typography.titleLarge)
                                Text(channel.name, maxLines = 2, style = MaterialTheme.typography.bodySmall)
                            }
                            val labelPaint = remember { TextPaint(Paint.ANTI_ALIAS_FLAG) }
                            Canvas(Modifier.weight(1f).fillMaxHeight()) {
                                val scale = 4.dp.toPx() / 60_000
                                val gap = 3.dp.toPx()
                                var selected = programs.indexOfFirst { selectedTime >= it.second && selectedTime < it.third }
                                if (selected < 0) selected = programs.indexOfFirst { it.second >= selectedTime }.let { if (it < 0) programs.lastIndex else it }
                                programs.forEachIndexed { programIndex, (program, start, end) ->
                                    val left = ((start - origin) * scale).coerceAtLeast(0f) + gap
                                    val right = ((end - origin) * scale).coerceAtMost(size.width)
                                    if (right <= left) return@forEachIndexed
                                    val current = now >= start && now < end
                                    val fill = if (current) palette.now else palette.surface
                                    drawRoundRect(fill, androidx.compose.ui.geometry.Offset(left, 0f), androidx.compose.ui.geometry.Size(right-left, size.height), androidx.compose.ui.geometry.CornerRadius(6.dp.toPx()))
                                    if (focused && programIndex == selected) drawRoundRect(Color.White, androidx.compose.ui.geometry.Offset(left+2, 2f), androidx.compose.ui.geometry.Size((right-left-4).coerceAtLeast(0f), size.height-4), androidx.compose.ui.geometry.CornerRadius(6.dp.toPx()), style = androidx.compose.ui.graphics.drawscope.Stroke(2.dp.toPx()))
                                    labelPaint.color = (if (current) palette.nowText else palette.text).toArgb()
                                    val available = (right-left-8.dp.toPx()).coerceAtLeast(0f)
                                    labelPaint.textSize = 11.dp.toPx()
                                    val label = (if (current) "Now · " else "") + format.format(Instant.ofEpochMilli(if (current) end else start))
                                    drawIntoCanvas { canvas ->
                                        canvas.nativeCanvas.drawText(TextUtils.ellipsize(label, labelPaint, available, TextUtils.TruncateAt.END).toString(), left+4.dp.toPx(), 17.dp.toPx(), labelPaint)
                                        labelPaint.textSize = 14.dp.toPx()
                                        canvas.nativeCanvas.drawText(TextUtils.ellipsize(program.title, labelPaint, available, TextUtils.TruncateAt.END).toString(), left+4.dp.toPx(), 39.dp.toPx(), labelPaint)
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
