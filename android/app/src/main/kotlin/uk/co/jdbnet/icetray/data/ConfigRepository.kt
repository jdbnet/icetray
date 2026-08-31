package uk.co.jdbnet.icetray.data

import android.content.Context
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import java.io.File
import java.util.UUID

class ConfigRepository(context: Context) {
    private val configDir = File(context.filesDir, "IceTray").apply { mkdirs() }
    private val configFile = File(configDir, "config.json")
    private val imagesDir = File(configDir, "images").apply { mkdirs() }
    private val json = Json {
        ignoreUnknownKeys = true
        prettyPrint = true
        encodeDefaults = true
    }

    suspend fun load(): AppConfig = withContext(Dispatchers.IO) {
        if (!configFile.exists()) {
            val default = AppConfig()
            save(default)
            return@withContext default
        }
        val config = json.decodeFromString<AppConfig>(configFile.readText())
        migrate(config)
    }

    suspend fun save(config: AppConfig) = withContext(Dispatchers.IO) {
        configFile.writeText(json.encodeToString(config))
    }

    fun imagesDirectory(): File = imagesDir

    fun imageFile(filename: String): File? {
        if (filename.isBlank()) return null
        return File(imagesDir, filename)
    }

    suspend fun getStreams(): List<StreamView> = withContext(Dispatchers.IO) {
        load().streams.map { it.toView() }
    }

    suspend fun addStream(name: String, url: String): StreamView = withContext(Dispatchers.IO) {
        val config = load()
        val stream = Stream(
            id = UUID.randomUUID().toString(),
            name = name.trim(),
            url = url.trim(),
        )
        save(config.copy(streams = config.streams + stream))
        stream.toView()
    }

    suspend fun updateStream(id: String, name: String, url: String) = withContext(Dispatchers.IO) {
        val config = load()
        val updated = config.streams.map {
            if (it.id == id) it.copy(name = name.trim(), url = url.trim()) else it
        }
        save(config.copy(streams = updated))
    }

    suspend fun removeStream(id: String): Stream? = withContext(Dispatchers.IO) {
        val config = load()
        val removed = config.streams.find { it.id == id } ?: return@withContext null
        val remaining = config.streams.filter { it.id != id }
        var next = config.copy(streams = remaining)
        if (next.lastStreamId == id) {
            next = next.copy(lastStreamId = "", lastStream = "")
        }
        save(next)
        removed
    }

    suspend fun reorderStreams(ids: List<String>) = withContext(Dispatchers.IO) {
        val config = load()
        if (ids.size != config.streams.size) {
            throw IllegalArgumentException("stream order length mismatch")
        }
        val byId = config.streams.associateBy { it.id }
        val seen = mutableSetOf<String>()
        val next = ids.map { id ->
            if (!seen.add(id)) {
                throw IllegalArgumentException("duplicate stream id: $id")
            }
            byId[id] ?: throw IllegalArgumentException("stream not found: $id")
        }
        save(config.copy(streams = next))
    }

    suspend fun setStreamImage(id: String, filename: String) = withContext(Dispatchers.IO) {
        val config = load()
        val updated = config.streams.map {
            if (it.id == id) it.copy(image = filename) else it
        }
        save(config.copy(streams = updated))
    }

    suspend fun setLastStreamId(id: String) = withContext(Dispatchers.IO) {
        val config = load()
        val stream = config.streams.find { it.id == id }
        save(
            config.copy(
                lastStreamId = id,
                lastStream = stream?.url ?: "",
            ),
        )
    }

    suspend fun setAutoplay(enabled: Boolean) = withContext(Dispatchers.IO) {
        save(load().copy(autoplay = enabled))
    }

    suspend fun setLaunchOnLogin(enabled: Boolean) = withContext(Dispatchers.IO) {
        save(load().copy(launchOnLogin = enabled))
    }

    suspend fun setVolume(volume: Int) = withContext(Dispatchers.IO) {
        save(load().copy(volume = volume.coerceIn(0, 100)))
    }

    suspend fun getSettings(): SettingsView = withContext(Dispatchers.IO) {
        val config = load()
        SettingsView(
            autoplay = config.autoplay,
            launchOnLogin = config.launchOnLogin,
            volume = config.volume,
        )
    }

    suspend fun getStreamById(id: String): Stream? = withContext(Dispatchers.IO) {
        load().streams.find { it.id == id }
    }

    private suspend fun migrate(config: AppConfig): AppConfig {
        var changed = false
        val streams = config.streams.map { stream ->
            if (stream.id.isBlank()) {
                changed = true
                stream.copy(id = UUID.randomUUID().toString())
            } else {
                stream
            }
        }
        var next = config.copy(streams = streams)
        if (next.lastStreamId.isBlank() && next.lastStream.isNotBlank()) {
            val match = streams.find { it.url == next.lastStream }
            if (match != null) {
                next = next.copy(lastStreamId = match.id)
                changed = true
            }
        }
        if (changed) {
            save(next)
        }
        return next
    }

    private fun Stream.toView(): StreamView {
        val path = imageFile(image)?.takeIf { it.exists() }?.absolutePath
        return StreamView(
            id = id,
            name = name,
            url = url,
            image = image,
            imagePath = path,
        )
    }
}
