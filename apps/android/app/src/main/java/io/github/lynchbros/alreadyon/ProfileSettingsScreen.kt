package io.github.lynchbros.alreadyon

import android.graphics.Bitmap
import android.graphics.ImageDecoder
import android.graphics.BitmapFactory
import androidx.compose.foundation.Image
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.layout.ContentScale
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.border
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.text.font.FontWeight
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONObject
import java.io.ByteArrayOutputStream

/** Web-equivalent settings cards keep drafts local until Save; PIN changes apply separately. */
@Composable
internal fun ProfileSettingsScreen(state: PhoneState, model: PhoneModel) {
    val profile = state.profile ?: return
    var name by remember(profile.id) { mutableStateOf(profile.name) }
    var icon by remember(profile.id) { mutableStateOf(profile.icon) }
    var upload by remember(profile.id) { mutableStateOf<ByteArray?>(null) }
    var time by remember(profile.id) { mutableStateOf(profile.timeFormat) }
    var audio by remember(profile.id) { mutableStateOf(profile.language) }
    var subtitles by remember(profile.id) { mutableStateOf(profile.subtitles) }
    var language by remember(profile.id) { mutableStateOf(profile.subtitleLanguage) }
    var editingPin by remember { mutableStateOf(false) }
    var removePin by remember { mutableStateOf(false) }
    var pin by remember { mutableStateOf("") }
    var confirm by remember { mutableStateOf("") }
    var localError by remember { mutableStateOf<String?>(null) }
    var processing by remember { mutableStateOf(false) }
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val picker = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        if (uri != null) scope.launch {
            processing = true
            try {
                upload = withContext(Dispatchers.IO) {
                    // Decode directly to a bounded image, then crop to the WebUI's 128px square.
                    val bitmap = if (android.os.Build.VERSION.SDK_INT >= 28) ImageDecoder.decodeBitmap(ImageDecoder.createSource(context.contentResolver, uri)) { decoder, info, _ ->
                        val scale = 256f / maxOf(info.size.width, info.size.height)
                        decoder.setTargetSize((info.size.width * scale).toInt().coerceAtLeast(1), (info.size.height * scale).toInt().coerceAtLeast(1))
                        decoder.allocator = ImageDecoder.ALLOCATOR_SOFTWARE
                    }
                    else {
                        val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
                        context.contentResolver.openInputStream(uri).use { BitmapFactory.decodeStream(it, null, bounds) }
                        require(bounds.outWidth > 0 && bounds.outHeight > 0)
                        val options = BitmapFactory.Options().apply { inSampleSize = (maxOf(bounds.outWidth, bounds.outHeight) / 256).coerceAtLeast(1) }
                        context.contentResolver.openInputStream(uri).use { requireNotNull(BitmapFactory.decodeStream(it, null, options)) }
                    }
                    val side = minOf(bitmap.width, bitmap.height)
                    val square = Bitmap.createBitmap(bitmap, (bitmap.width - side) / 2, (bitmap.height - side) / 2, side, side)
                    val small = Bitmap.createScaledBitmap(square, 128, 128, true)
                    ByteArrayOutputStream().use { bytes -> small.compress(Bitmap.CompressFormat.PNG, 100, bytes); bytes.toByteArray() }
                }
                localError = null
            } catch (e: kotlinx.coroutines.CancellationException) { throw e }
            catch (_: Exception) { localError = "Couldn’t read that image." }
            finally { processing = false }
        }
    }
    val changed = name.trim() != profile.name || icon != profile.icon || upload != null || time != profile.timeFormat ||
        audio != profile.language || subtitles != profile.subtitles || language != profile.subtitleLanguage
    LazyColumn(Modifier.fillMaxWidth(), contentPadding = PaddingValues(12.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        item { OutlinedButton(shape = MaterialTheme.shapes.small, onClick = model::closeSettings) { Text("← Back To Guide") } }
        item { SettingsCard("Display Name") { OutlinedTextField(name, { name = it }, label = { Text("Name") }, singleLine = true, modifier = Modifier.fillMaxWidth()) } }
        item { SettingsCard("Icon") {
            if (upload == null) ProfileName(profile.copy(name = name, icon = icon), model, avatarSize = 64.dp, showName = false)
            upload?.let { bytes ->
                val bitmap = remember(bytes) { BitmapFactory.decodeByteArray(bytes, 0, bytes.size) }
                Image(bitmap.asImageBitmap(), "Custom icon preview", Modifier.size(64.dp), contentScale = ContentScale.Crop)
            }
            Row(Modifier.horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedButton(shape = MaterialTheme.shapes.small, onClick = { picker.launch("image/*") }, enabled = !processing) { Text("Upload Custom Icon") }
                OutlinedButton(shape = MaterialTheme.shapes.small, onClick = { icon = null; upload = null }) { Text("Remove Icon") }
            }
            // Wrap presets to the available width so every icon remains visible on phones.
            FlowRow(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp)) {
                avatarPresets.forEach { (id, preset) ->
                    FilledTonalButton(onClick = { icon = id; upload = null }, contentPadding = PaddingValues(0.dp),
                        modifier = Modifier.size(48.dp).then(if (icon == id && upload == null) Modifier.border(2.dp, Color(0xff389bea), MaterialTheme.shapes.extraLarge) else Modifier),
                        colors = ButtonDefaults.filledTonalButtonColors(containerColor = Color(preset.second))) { Text(preset.first) }
                }
            }
        } }
        item { SettingsCard("PIN") {
            Text("Status   ${if (profile.hasPin) "PIN Is Set" else "No PIN Set"}")
            if (editingPin) {
                OutlinedTextField(pin, { pin = it.filter(Char::isDigit).take(6) }, label = { Text("New PIN (4-6 Digits)") },
                    visualTransformation = androidx.compose.ui.text.input.PasswordVisualTransformation(),
                    keyboardOptions = androidx.compose.foundation.text.KeyboardOptions(keyboardType = androidx.compose.ui.text.input.KeyboardType.NumberPassword))
                OutlinedTextField(confirm, { confirm = it.filter(Char::isDigit).take(6) }, label = { Text("Confirm PIN") },
                    visualTransformation = androidx.compose.ui.text.input.PasswordVisualTransformation(),
                    keyboardOptions = androidx.compose.foundation.text.KeyboardOptions(keyboardType = androidx.compose.ui.text.input.KeyboardType.NumberPassword))
                Row {
                    Button(onClick = { model.savePin(pin); pin = ""; confirm = ""; editingPin = false }, enabled = !state.busy && pin.length in 4..6 && pin == confirm) { Text("Save PIN") }
                    TextButton(onClick = { editingPin = false; pin = ""; confirm = "" }) { Text("Cancel") }
                }
            } else Row(Modifier.horizontalScroll(rememberScrollState())) {
                OutlinedButton(shape = MaterialTheme.shapes.small, onClick = { editingPin = true; removePin = false }) { Text(if (profile.hasPin) "Change PIN" else "Set PIN") }
                if (profile.hasPin) OutlinedButton(shape = MaterialTheme.shapes.small, onClick = { if (removePin) { model.savePin(null); removePin = false } else removePin = true }, enabled = !state.busy) { Text(if (removePin) "Confirm remove?" else "Remove PIN") }
            }
        } }
        item { SettingsCard("Guide Display") { SettingsChoice("Time Format", time, listOf("12h-padded" to "01:00am-02:00am", "12h" to "1:00am-2:00am", "24h" to "22:00-23:00")) { time = it } } }
        item { SettingsCard("Audio") { SettingsChoice("Preferred Audio Language", audio, profileLanguages) { audio = it } } }
        item { SettingsCard("Subtitles") {
            SettingsChoice("Subtitles By Default", if (subtitles) "on" else "off", listOf("off" to "Disabled", "on" to "Enabled")) { subtitles = it == "on" }
            SettingsChoice("Default Language", language, profileLanguages) { language = it }
        } }
        localError?.let { item { Text(it, color = MaterialTheme.colorScheme.error) } }
        state.notice?.let { item { Text(it) } }
        item { Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Button(onClick = {
                val patch = JSONObject()
                if (name.trim() != profile.name) patch.put("name", name.trim())
                if (upload == null && icon != profile.icon) patch.put("icon", icon ?: JSONObject.NULL)
                if (time != profile.timeFormat) patch.put("guideTimeFormat", time)
                if (audio != profile.language) patch.put("audioLanguage", audio)
                if (subtitles != profile.subtitles) patch.put("subtitlesEnabled", subtitles)
                if (language != profile.subtitleLanguage) patch.put("subtitleLanguage", language)
                model.saveProfile(patch, upload)
            }, enabled = changed && !state.busy && !processing && name.isNotBlank()) { Text("Save Changes") }
            OutlinedButton(shape = MaterialTheme.shapes.small, onClick = model::closeSettings) { Text("Cancel") }
        } }
    }
}

/** Cards share the native dark palette and adapt the web fields to phone width. */
@Composable
private fun SettingsCard(title: String, content: @Composable ColumnScope.() -> Unit) {
    Surface(color = phonePalette.surface, shape = MaterialTheme.shapes.medium,
        modifier = Modifier.fillMaxWidth().border(1.dp, phonePalette.border, MaterialTheme.shapes.medium)) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Text(title, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
            content()
        }
    }
}

@Composable
private fun SettingsChoice(label: String, value: String, options: List<Pair<String, String>>, changed: (String) -> Unit) {
    var open by remember { mutableStateOf(false) }
    Text(label, color = phonePalette.muted)
    Box {
        OutlinedButton(shape = MaterialTheme.shapes.small, onClick = { open = true }, modifier = Modifier.fillMaxWidth()) { Text((options.find { it.first == value }?.second ?: value) + " ▾") }
        DropdownMenu(expanded = open, onDismissRequest = { open = false }) {
            options.forEach { (code, name) -> DropdownMenuItem(text = { Text(name) }, onClick = { changed(code); open = false }) }
        }
    }
}

private val profileLanguages = listOf("en", "es", "fr", "de", "it", "pt", "ja", "ko", "zh", "ru", "nl", "sv", "da", "no", "fi", "pl", "tr", "ar", "he", "hi", "cs", "el", "hu", "th", "uk")
    .map { it to java.util.Locale.forLanguageTag(it).getDisplayLanguage(java.util.Locale.ENGLISH) }
