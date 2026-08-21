package uk.co.jdbnet.icetray.ui

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import uk.co.jdbnet.icetray.IceTrayApp
import uk.co.jdbnet.icetray.data.NowPlaying
import uk.co.jdbnet.icetray.data.PlaybackState
import uk.co.jdbnet.icetray.data.SettingsView
import uk.co.jdbnet.icetray.data.StreamView
import uk.co.jdbnet.icetray.playback.PlaybackController

class PlayerViewModel(application: Application) : AndroidViewModel(application) {
    private val app = IceTrayApp.get(application)
    private val configRepository = app.configRepository
    private val imageStore = app.imageStore

    private val _streams = MutableStateFlow<List<StreamView>>(emptyList())
    val streams: StateFlow<List<StreamView>> = _streams.asStateFlow()

    private val _settings = MutableStateFlow(SettingsView())
    val settings: StateFlow<SettingsView> = _settings.asStateFlow()

    private val _loading = MutableStateFlow(true)
    val loading: StateFlow<Boolean> = _loading.asStateFlow()

    val playbackState: StateFlow<PlaybackState> = PlaybackController.playbackState
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), PlaybackState())

    val nowPlaying: StateFlow<NowPlaying> = PlaybackController.nowPlaying
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), NowPlaying())

    init {
        viewModelScope.launch {
            refreshAll()
            maybeAutoplay()
        }
    }

    suspend fun refreshAll() {
        _loading.value = true
        _streams.value = configRepository.getStreams()
        _settings.value = configRepository.getSettings()
        _loading.value = false
    }

    fun refreshStreams() {
        viewModelScope.launch {
            _streams.value = configRepository.getStreams()
        }
    }

    private suspend fun maybeAutoplay() {
        val settings = configRepository.getSettings()
        if (!settings.autoplay) return
        val config = configRepository.load()
        val id = config.lastStreamId.ifBlank {
            config.streams.find { it.url == config.lastStream }?.id.orEmpty()
        }
        if (id.isBlank()) return
        val stream = _streams.value.find { it.id == id } ?: return
        playStream(stream)
    }

    fun playStream(stream: StreamView) {
        viewModelScope.launch {
            configRepository.setLastStreamId(stream.id)
            val volume = configRepository.getSettings().volume
            PlaybackController.play(getApplication(), stream, volume)
        }
    }

    fun togglePlay(currentStream: StreamView?) {
        val state = playbackState.value
        when {
            state.playing -> PlaybackController.pause()
            state.paused -> PlaybackController.resume()
            currentStream != null -> playStream(currentStream)
            streams.value.isNotEmpty() -> playStream(streams.value.first())
        }
    }

    fun stopPlayback() {
        PlaybackController.stop(getApplication())
    }

    fun setVolume(volume: Int) {
        viewModelScope.launch {
            configRepository.setVolume(volume)
            PlaybackController.setVolume(volume)
            _settings.value = configRepository.getSettings()
        }
    }

    fun setAutoplay(enabled: Boolean) {
        viewModelScope.launch {
            configRepository.setAutoplay(enabled)
            _settings.value = configRepository.getSettings()
        }
    }

    fun setLaunchOnLogin(enabled: Boolean) {
        viewModelScope.launch {
            configRepository.setLaunchOnLogin(enabled)
            _settings.value = configRepository.getSettings()
        }
    }

    fun addStream(name: String, url: String, onError: (String) -> Unit, onSuccess: () -> Unit = {}) {
        viewModelScope.launch {
            runCatching {
                if (name.isBlank() || url.isBlank()) {
                    throw IllegalArgumentException("name and URL are required")
                }
                configRepository.addStream(name, url)
            }.onSuccess {
                refreshStreams()
                onSuccess()
            }.onFailure {
                onError(it.message ?: "Failed to save stream")
            }
        }
    }

    fun updateStream(id: String, name: String, url: String, onError: (String) -> Unit, onSuccess: () -> Unit = {}) {
        viewModelScope.launch {
            runCatching {
                if (name.isBlank() || url.isBlank()) {
                    throw IllegalArgumentException("name and URL are required")
                }
                configRepository.updateStream(id, name, url)
            }.onSuccess {
                refreshStreams()
                onSuccess()
            }.onFailure {
                onError(it.message ?: "Failed to save stream")
            }
        }
    }

    fun removeStream(id: String) {
        viewModelScope.launch {
            val removed = configRepository.removeStream(id)
            if (removed?.image?.isNotBlank() == true) {
                imageStore.deleteImage(removed.image)
            }
            refreshStreams()
        }
    }

    fun uploadImage(streamId: String, uri: android.net.Uri, onError: (String) -> Unit) {
        viewModelScope.launch {
            runCatching {
                val filename = imageStore.saveStreamImage(streamId, uri, getApplication())
                configRepository.setStreamImage(streamId, filename)
            }.onSuccess {
                refreshStreams()
            }.onFailure {
                onError(it.message ?: "Failed to save image")
            }
        }
    }
}
