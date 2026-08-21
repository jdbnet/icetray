package uk.co.jdbnet.icetray.metadata

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.intOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import uk.co.jdbnet.icetray.data.NowPlaying
import java.io.BufferedInputStream
import java.net.HttpURLConnection
import java.net.URL

class MetadataFetcher {
    private val json = Json { ignoreUnknownKeys = true }

    suspend fun fetch(streamUrl: String): NowPlaying = withContext(Dispatchers.IO) {
        val (base, mount) = parseStreamUrl(streamUrl)
        fetchPublicStats(base, mount)
            ?: fetchLegacyStats(base, mount)
            ?: fetchIcyMetadata(streamUrl)
            ?: throw IllegalStateException("no metadata available")
    }

    private fun fetchPublicStats(base: String, mount: String): NowPlaying? {
        val body = fetchUrl("$base/admin/publicstats.json") ?: return null
        val docs = runCatching { json.parseToJsonElement(body).jsonArray }.getOrNull() ?: return null
        for (doc in docs) {
            val source = doc.jsonObject["source"] ?: continue
            val entries = parseSourceEntries(source)
            matchSource(entries, mount)?.let { return it }
        }
        return null
    }

    private fun fetchLegacyStats(base: String, mount: String): NowPlaying? {
        val body = fetchUrl("$base/status-json.xsl") ?: return null
        val root = runCatching { json.parseToJsonElement(body).jsonObject }.getOrNull() ?: return null
        val source = root["icestats"]?.jsonObject?.get("source") ?: return null
        val entries = parseSourceEntries(source)
        if (entries.isEmpty()) return null
        return matchSource(entries, mount)
    }

    private fun fetchIcyMetadata(streamUrl: String): NowPlaying? {
        val connection = openConnection(streamUrl) ?: return null
        connection.setRequestProperty("Icy-MetaData", "1")
        connection.connect()
        if (connection.responseCode != HttpURLConnection.HTTP_OK) {
            connection.disconnect()
            return null
        }
        val station = listOf("Icy-Name", "Ice-Name", "icy-name")
            .firstNotNullOfOrNull { connection.getHeaderField(it) }
            .orEmpty()
        val genre = listOf("Icy-Genre", "Ice-Genre", "icy-genre")
            .firstNotNullOfOrNull { connection.getHeaderField(it) }
            .orEmpty()
        val metaIntHeader = listOf("Icy-MetaInt", "icy-metaint")
            .firstNotNullOfOrNull { connection.getHeaderField(it) }
        val title = metaIntHeader?.toIntOrNull()?.let { blockSize ->
            connection.inputStream.use { input ->
                readFirstIcyTitle(BufferedInputStream(input), blockSize)
            }
        }.orEmpty()
        connection.disconnect()
        if (title.isBlank() && station.isBlank()) return null
        return NowPlaying(station = station, title = title, genre = genre)
    }

    private fun readFirstIcyTitle(input: BufferedInputStream, blockSize: Int): String {
        val audio = ByteArray(blockSize)
        if (input.read(audio) != blockSize) return ""
        val metaLenByte = input.read()
        if (metaLenByte <= 0) return ""
        val metaLen = metaLenByte * 16
        val meta = ByteArray(metaLen)
        if (input.read(meta) != metaLen) return ""
        return parseStreamTitle(String(meta, Charsets.ISO_8859_1))
    }

    private fun parseStreamTitle(meta: String): String {
        for (part in meta.split(';')) {
            val trimmed = part.trim()
            if (trimmed.isEmpty()) continue
            val idx = trimmed.indexOf('=')
            if (idx <= 0) continue
            val key = trimmed.substring(0, idx).trim()
            if (!key.equals("StreamTitle", ignoreCase = true)) continue
            return trimmed.substring(idx + 1).trim().trim('\'', '"', ' ')
        }
        return ""
    }

    private data class SourceEntry(val mount: String = "", val data: JsonObject)

    private fun parseSourceEntries(source: JsonElement): List<SourceEntry> {
        if (source is JsonArray) {
            return source.mapNotNull { item ->
                (item as? JsonObject)?.let { SourceEntry(data = it) }
            }
        }
        val obj = source as? JsonObject ?: return emptyList()
        if (obj.containsKey("listenurl")) {
            return listOf(SourceEntry(data = obj))
        }
        return obj.mapNotNull { (mount, value) ->
            (value as? JsonObject)?.let { SourceEntry(mount = mount, data = it) }
        }
    }

    private fun matchSource(entries: List<SourceEntry>, mount: String): NowPlaying? {
        val normalizedMount = normalizeMount(mount)
        var fallback: NowPlaying? = null
        for (entry in entries) {
            val np = entryToNowPlaying(entry.data)
            if (np.station.isBlank() && np.title.isBlank() && np.genre.isBlank() && np.listeners == 0) {
                continue
            }
            if (entry.mount.isNotBlank() && mountsEqual(entry.mount, normalizedMount)) {
                return np
            }
            val listenUrl = stringField(entry.data, "listenurl", "ListenURL")
            if (listenUrl.isNotBlank() && sourceMatches(listenUrl, normalizedMount)) {
                return np
            }
            if (fallback == null) {
                fallback = np
            }
        }
        return if (entries.size == 1) fallback else null
    }

    private fun entryToNowPlaying(data: JsonObject): NowPlaying {
        val station = firstStringField(data, "server_name", "server_description", "ServerName")
            .ifBlank { stringField(data, "server_type", "ServerType") }
        return NowPlaying(
            station = station,
            title = extractTitle(data),
            genre = stringField(data, "genre", "Genre"),
            listeners = intField(data, "listeners", "Listeners"),
        )
    }

    private fun extractTitle(data: JsonObject): String {
        for (key in listOf("display-title", "display_title", "title", "Title")) {
            stringField(data, key).takeIf { it.isNotBlank() }?.let { return it }
        }
        val meta = data["metadata"] as? JsonObject
        if (meta != null) {
            for (key in listOf("streamtitle", "StreamTitle", "x_icy_title", "icy-title", "title", "Title")) {
                stringField(meta, key).takeIf { it.isNotBlank() }?.let { return it }
            }
        }
        val playlist = data["playlist"] as? JsonObject
        if (playlist != null) {
            lastPlaylistTitle(playlist)?.takeIf { it.isNotBlank() }?.let { return it }
        }
        return ""
    }

    private fun lastPlaylistTitle(playlist: JsonObject): String? {
        val inner = playlist["playlist"] as? JsonObject ?: return null
        val tracks = inner["track"] as? JsonArray ?: return null
        if (tracks.isEmpty()) return null
        val last = tracks.last() as? JsonObject ?: return null
        return stringField(last, "title", "Title").ifBlank { null }
    }

    private fun stringField(obj: JsonObject, vararg keys: String): String {
        for (key in keys) {
            val value = obj[key]?.jsonPrimitive?.contentOrNull
            if (!value.isNullOrBlank()) return value
        }
        return ""
    }

    private fun firstStringField(obj: JsonObject, vararg keys: String): String = stringField(obj, *keys)

    private fun intField(obj: JsonObject, vararg keys: String): Int {
        for (key in keys) {
            val primitive = obj[key]?.jsonPrimitive ?: continue
            primitive.intOrNull?.let { return it }
            primitive.contentOrNull?.toIntOrNull()?.let { return it }
        }
        return 0
    }

    private fun fetchUrl(endpoint: String): String? {
        val connection = openConnection(endpoint) ?: return null
        connection.connect()
        if (connection.responseCode != HttpURLConnection.HTTP_OK) {
            connection.disconnect()
            return null
        }
        val body = connection.inputStream.bufferedReader().use { it.readText().take(1_000_000) }
        connection.disconnect()
        return body
    }

    private fun openConnection(url: String): HttpURLConnection? {
        return runCatching {
            (URL(url).openConnection() as HttpURLConnection).apply {
                connectTimeout = 10_000
                readTimeout = 12_000
                instanceFollowRedirects = true
            }
        }.getOrNull()
    }

    private fun parseStreamUrl(streamUrl: String): Pair<String, String> {
        val url = URL(streamUrl)
        var mount = url.path
        if (mount.isNullOrBlank()) mount = "/"
        val base = URL(url.protocol, url.host, url.port, "").toString().trimEnd('/')
        return base to mount
    }

    private fun normalizeMount(mount: String): String {
        val trimmed = mount.trim()
        if (trimmed.isEmpty()) return "/"
        return if (trimmed.startsWith("/")) trimmed else "/$trimmed"
    }

    private fun mountsEqual(a: String, b: String): Boolean = normalizeMount(a) == normalizeMount(b)

    private fun sourceMatches(listenUrl: String, mount: String): Boolean {
        val path = runCatching { URL(listenUrl).path }.getOrNull() ?: return false
        val normalized = normalizeMount(path)
        return normalized == mount || normalized.endsWith(mount, ignoreCase = true)
    }
}
