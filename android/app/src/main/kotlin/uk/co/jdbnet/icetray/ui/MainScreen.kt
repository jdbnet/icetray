package uk.co.jdbnet.icetray.ui

import android.Manifest
import android.net.Uri
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.Image
import androidx.compose.material.icons.filled.Login
import androidx.compose.material.icons.filled.MusicNote
import androidx.compose.material.icons.filled.Pause
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.Repeat
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.filled.Stop
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.Composable
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.lifecycle.viewmodel.compose.viewModel
import coil.compose.AsyncImage
import uk.co.jdbnet.icetray.data.StreamView
import uk.co.jdbnet.icetray.ui.IceTrayColors.Background
import uk.co.jdbnet.icetray.ui.IceTrayColors.Border
import uk.co.jdbnet.icetray.ui.IceTrayColors.Emerald
import uk.co.jdbnet.icetray.ui.IceTrayColors.Surface
import uk.co.jdbnet.icetray.ui.IceTrayColors.Zinc100
import uk.co.jdbnet.icetray.ui.IceTrayColors.Zinc300
import uk.co.jdbnet.icetray.ui.IceTrayColors.Zinc400
import uk.co.jdbnet.icetray.ui.IceTrayColors.Zinc500
import uk.co.jdbnet.icetray.ui.IceTrayColors.Zinc800
import uk.co.jdbnet.icetray.ui.IceTrayColors.Zinc900

private val IceTrayDarkScheme = darkColorScheme(
    primary = Emerald,
    onPrimary = Color.Black,
    background = Background,
    surface = Zinc900,
    onBackground = Zinc100,
    onSurface = Zinc100,
)

@Composable
fun IceTrayTheme(content: @Composable () -> Unit) {
    MaterialTheme(colorScheme = IceTrayDarkScheme, content = content)
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MainScreen(viewModel: PlayerViewModel = viewModel()) {
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
        val permissionLauncher = rememberLauncherForActivityResult(
            contract = ActivityResultContracts.RequestPermission(),
        ) { }
        LaunchedEffect(Unit) {
            permissionLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
        }
    }

    val streams by viewModel.streams.collectAsState()
    val settings by viewModel.settings.collectAsState()
    val loading by viewModel.loading.collectAsState()
    val playback by viewModel.playbackState.collectAsState()
    val nowPlaying by viewModel.nowPlaying.collectAsState()

    var showSettings by remember { mutableStateOf(false) }
    var showModal by remember { mutableStateOf(false) }
    var editing by remember { mutableStateOf<StreamView?>(null) }
    var formName by remember { mutableStateOf("") }
    var formUrl by remember { mutableStateOf("") }
    var formError by remember { mutableStateOf("") }
    var artworkTargetId by remember { mutableStateOf<String?>(null) }

    val imagePicker = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.PickVisualMedia(),
    ) { uri: Uri? ->
        val id = artworkTargetId
        if (uri != null && id != null) {
            viewModel.uploadImage(id, uri) { formError = it }
        }
        artworkTargetId = null
    }

    val currentStream = streams.find { it.id == playback.streamId }
    val displayTitle = nowPlaying.title.ifBlank {
        currentStream?.name ?: "Nothing playing"
    }
    val displaySubtitle = when {
        nowPlaying.station.isNotBlank() && nowPlaying.title.isNotBlank() -> nowPlaying.station
        nowPlaying.genre.isNotBlank() -> nowPlaying.genre
        currentStream != null -> currentStream.url
        else -> "Select a stream to begin"
    }

    Scaffold(
        containerColor = Background,
        bottomBar = {
            NowPlayingBar(
                imagePath = currentStream?.imagePath,
                title = displayTitle,
                subtitle = displaySubtitle,
                listeners = nowPlaying.listeners,
                playing = playback.playing,
                onTogglePlay = { viewModel.togglePlay(currentStream) },
                onStop = viewModel::stopPlayback,
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .background(
                    Brush.radialGradient(
                        colors = listOf(Color(0xFF1A1F2E), Background),
                        radius = 1200f,
                    ),
                ),
        ) {
            Header(
                showSettings = showSettings,
                onToggleSettings = { showSettings = !showSettings },
                onAdd = {
                    editing = null
                    formName = ""
                    formUrl = ""
                    formError = ""
                    showModal = true
                },
            )

            if (showSettings) {
                SettingsRow(
                    autoplay = settings.autoplay,
                    launchOnLogin = settings.launchOnLogin,
                    onToggleAutoplay = { viewModel.setAutoplay(!settings.autoplay) },
                    onToggleLaunchOnLogin = { viewModel.setLaunchOnLogin(!settings.launchOnLogin) },
                )
            }

            when {
                loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator(color = Emerald)
                }

                streams.isEmpty() -> EmptyState(onAdd = {
                    editing = null
                    formName = ""
                    formUrl = ""
                    formError = ""
                    showModal = true
                })

                else -> LazyVerticalGrid(
                    columns = GridCells.Adaptive(160.dp),
                    contentPadding = PaddingValues(16.dp),
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                    modifier = Modifier.fillMaxSize(),
                ) {
                    items(streams, key = { it.id }) { stream ->
                        StreamCard(
                            stream = stream,
                            selected = playback.streamId == stream.id,
                            onPlay = { viewModel.playStream(stream) },
                            onEdit = {
                                editing = stream
                                formName = stream.name
                                formUrl = stream.url
                                formError = ""
                                showModal = true
                            },
                            onDelete = { viewModel.removeStream(stream.id) },
                            onArtwork = {
                                artworkTargetId = stream.id
                                imagePicker.launch(
                                    PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly),
                                )
                            },
                        )
                    }
                }
            }
        }
    }

    if (showModal) {
        StreamDialog(
            editing = editing,
            name = formName,
            url = formUrl,
            error = formError,
            onNameChange = { formName = it },
            onUrlChange = { formUrl = it },
            onDismiss = { showModal = false },
            onSave = {
                formError = ""
                if (editing != null) {
                    viewModel.updateStream(editing!!.id, formName, formUrl,
                        onError = { formError = it },
                        onSuccess = { showModal = false },
                    )
                } else {
                    viewModel.addStream(formName, formUrl,
                        onError = { formError = it },
                        onSuccess = { showModal = false },
                    )
                }
            },
        )
    }
}

@Composable
private fun Header(showSettings: Boolean, onToggleSettings: () -> Unit, onAdd: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .border(width = 1.dp, color = Border)
            .padding(horizontal = 20.dp, vertical = 16.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column {
            Text("IceTray", style = MaterialTheme.typography.titleLarge, color = Zinc100)
            Text("Your Icecast stations", style = MaterialTheme.typography.bodySmall, color = Zinc400)
        }
        Row(horizontalArrangement = Arrangement.spacedBy(16.dp)) {
            CircleIconButton(
                active = showSettings,
                onClick = onToggleSettings,
                contentDescription = "Settings",
            ) {
                Icon(Icons.Default.Settings, contentDescription = null)
            }
            CircleIconButton(primary = true, onClick = onAdd, contentDescription = "Add stream") {
                Icon(Icons.Default.Add, contentDescription = null, tint = Color.Black)
            }
        }
    }
}

@Composable
private fun SettingsRow(
    autoplay: Boolean,
    launchOnLogin: Boolean,
    onToggleAutoplay: () -> Unit,
    onToggleLaunchOnLogin: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(Surface)
            .border(width = 1.dp, color = Border)
            .padding(16.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        SettingToggle(active = autoplay, onClick = onToggleAutoplay, description = "Autoplay on startup") {
            Icon(Icons.Default.Repeat, contentDescription = null, modifier = Modifier.size(18.dp))
        }
        SettingToggle(active = launchOnLogin, onClick = onToggleLaunchOnLogin, description = "Resume on boot") {
            Icon(Icons.Default.Login, contentDescription = null, modifier = Modifier.size(18.dp))
        }
    }
}

@Composable
private fun EmptyState(onAdd: () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Icon(Icons.Default.MusicNote, contentDescription = null, modifier = Modifier.size(48.dp), tint = Zinc500)
        Spacer(Modifier.height(12.dp))
        Text("No streams yet", style = MaterialTheme.typography.titleMedium, color = Zinc300)
        Text("Add your first Icecast stream to get started.", color = Zinc500)
        Spacer(Modifier.height(16.dp))
        CircleIconButton(primary = true, onClick = onAdd, contentDescription = "Add stream") {
            Icon(Icons.Default.Add, contentDescription = null, tint = Color.Black)
        }
    }
}

@Composable
private fun StreamCard(
    stream: StreamView,
    selected: Boolean,
    onPlay: () -> Unit,
    onEdit: () -> Unit,
    onDelete: () -> Unit,
    onArtwork: () -> Unit,
) {
    Box(
        modifier = Modifier
            .clip(RoundedCornerShape(16.dp))
            .background(Surface)
            .border(
                width = if (selected) 2.dp else 1.dp,
                color = if (selected) Emerald.copy(alpha = 0.6f) else Border,
                shape = RoundedCornerShape(16.dp),
            )
            .clickable(onClick = onPlay),
    ) {
        Column {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .aspectRatio(1f)
                    .background(Zinc900),
                contentAlignment = Alignment.Center,
            ) {
                if (stream.imagePath != null) {
                    AsyncImage(
                        model = stream.imagePath,
                        contentDescription = stream.name,
                        contentScale = ContentScale.Crop,
                        modifier = Modifier.fillMaxSize(),
                    )
                } else {
                    Icon(Icons.Default.MusicNote, contentDescription = null, modifier = Modifier.size(40.dp), tint = Zinc500)
                }
            }
            Column(modifier = Modifier.padding(12.dp)) {
                Text(stream.name, maxLines = 1, overflow = TextOverflow.Ellipsis, color = Zinc100)
                Text(stream.url, maxLines = 1, overflow = TextOverflow.Ellipsis, style = MaterialTheme.typography.bodySmall, color = Zinc500)
            }
        }
        Row(
            modifier = Modifier
                .align(Alignment.TopEnd)
                .padding(8.dp),
            horizontalArrangement = Arrangement.spacedBy(4.dp),
        ) {
            SmallActionIcon(Icons.Default.Image, "Upload artwork", onArtwork)
            SmallActionIcon(Icons.Default.Edit, "Edit stream", onEdit)
            SmallActionIcon(Icons.Default.Delete, "Delete stream", onDelete)
        }
    }
}

@Composable
private fun NowPlayingBar(
    imagePath: String?,
    title: String,
    subtitle: String,
    listeners: Int,
    playing: Boolean,
    onTogglePlay: () -> Unit,
    onStop: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(Color(0x66000000))
            .border(width = 1.dp, color = Border)
            .padding(horizontal = 16.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Box(
            modifier = Modifier
                .size(56.dp)
                .clip(RoundedCornerShape(8.dp))
                .background(Zinc800),
            contentAlignment = Alignment.Center,
        ) {
            if (imagePath != null) {
                AsyncImage(model = imagePath, contentDescription = null, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
            } else {
                Icon(Icons.Default.MusicNote, contentDescription = null, tint = Zinc500)
            }
        }
        Column(modifier = Modifier.weight(1f)) {
            Text(title, maxLines = 2, overflow = TextOverflow.Ellipsis, color = Zinc100)
            Text(subtitle, maxLines = 2, overflow = TextOverflow.Ellipsis, color = Zinc400, style = MaterialTheme.typography.bodySmall)
            if (listeners > 0) {
                Text("$listeners listeners", color = Zinc500, style = MaterialTheme.typography.labelSmall)
            }
        }
        CircleIconButton(onClick = onTogglePlay, contentDescription = if (playing) "Pause" else "Play") {
            Icon(if (playing) Icons.Default.Pause else Icons.Default.PlayArrow, contentDescription = null)
        }
        CircleIconButton(onClick = onStop, contentDescription = "Stop") {
            Icon(Icons.Default.Stop, contentDescription = null)
        }
    }
}

@Composable
private fun StreamDialog(
    editing: StreamView?,
    name: String,
    url: String,
    error: String,
    onNameChange: (String) -> Unit,
    onUrlChange: (String) -> Unit,
    onDismiss: () -> Unit,
    onSave: () -> Unit,
) {
    Dialog(onDismissRequest = onDismiss) {
        Column(
            modifier = Modifier
                .clip(RoundedCornerShape(16.dp))
                .background(Zinc900)
                .border(1.dp, Border, RoundedCornerShape(16.dp))
                .padding(20.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    if (editing == null) "Add stream" else "Edit stream",
                    style = MaterialTheme.typography.titleMedium,
                    color = Zinc100,
                )
                IconButton(onClick = onDismiss) {
                    Icon(Icons.Default.Close, contentDescription = "Close", tint = Zinc100)
                }
            }
            Spacer(Modifier.height(12.dp))
            OutlinedTextField(value = name, onValueChange = onNameChange, label = { Text("Station name") }, modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(8.dp))
            OutlinedTextField(value = url, onValueChange = onUrlChange, label = { Text("Stream URL") }, modifier = Modifier.fillMaxWidth())
            if (error.isNotBlank()) {
                Spacer(Modifier.height(8.dp))
                Text(error, color = Color(0xFFF87171))
            }
            Row(modifier = Modifier.fillMaxWidth().padding(top = 16.dp), horizontalArrangement = Arrangement.End) {
                TextButton(onClick = onDismiss) {
                    Icon(Icons.Default.Close, contentDescription = "Cancel", tint = Zinc100)
                }
                Spacer(Modifier.width(8.dp))
                CircleIconButton(primary = true, onClick = onSave, contentDescription = "Save") {
                    Icon(Icons.Default.Check, contentDescription = null, tint = Color.Black)
                }
            }
        }
    }
}

@Composable
private fun CircleIconButton(
    onClick: () -> Unit,
    contentDescription: String,
    active: Boolean = false,
    primary: Boolean = false,
    content: @Composable () -> Unit,
) {
    val bg = when {
        primary -> Emerald
        active -> Emerald.copy(alpha = 0.15f)
        else -> Surface
    }
    val borderColor = when {
        primary -> Emerald.copy(alpha = 0.4f)
        active -> Emerald.copy(alpha = 0.5f)
        else -> Border
    }
    Box(
        modifier = Modifier
            .size(40.dp)
            .semantics { this.contentDescription = contentDescription }
            .clip(CircleShape)
            .background(bg)
            .border(1.dp, borderColor, CircleShape)
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        content()
    }
}

@Composable
private fun SettingToggle(
    active: Boolean,
    onClick: () -> Unit,
    description: String,
    content: @Composable () -> Unit,
) {
    IconButton(
        onClick = onClick,
        modifier = Modifier
            .size(44.dp)
            .background(if (active) Emerald.copy(alpha = 0.15f) else Color(0x33000000), RoundedCornerShape(8.dp))
            .border(1.dp, if (active) Emerald.copy(alpha = 0.5f) else Border, RoundedCornerShape(8.dp)),
    ) {
        content()
    }
}

@Composable
private fun SmallActionIcon(
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    description: String,
    onClick: () -> Unit,
) {
    IconButton(
        onClick = onClick,
        modifier = Modifier
            .size(32.dp)
            .background(Color(0xA6000000), RoundedCornerShape(6.dp)),
    ) {
        Icon(icon, contentDescription = description, modifier = Modifier.size(16.dp), tint = Zinc300)
    }
}
