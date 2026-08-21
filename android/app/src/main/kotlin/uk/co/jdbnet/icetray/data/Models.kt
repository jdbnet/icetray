package uk.co.jdbnet.icetray.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class Stream(
    val id: String = "",
    val name: String = "",
    val url: String = "",
    val image: String = "",
)

@Serializable
data class AppConfig(
    val streams: List<Stream> = emptyList(),
    @SerialName("last_stream") val lastStream: String = "",
    @SerialName("last_stream_id") val lastStreamId: String = "",
    val autoplay: Boolean = false,
    val volume: Int = 50,
    @SerialName("launch_on_login") val launchOnLogin: Boolean = false,
)

data class StreamView(
    val id: String,
    val name: String,
    val url: String,
    val image: String = "",
    val imagePath: String? = null,
)

data class PlaybackState(
    val playing: Boolean = false,
    val paused: Boolean = false,
    val streamId: String = "",
    val volume: Int = 50,
)

data class SettingsView(
    val autoplay: Boolean = false,
    val launchOnLogin: Boolean = false,
    val volume: Int = 50,
)

data class NowPlaying(
    val station: String = "",
    val title: String = "",
    val genre: String = "",
    val listeners: Int = 0,
)
