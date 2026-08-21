package uk.co.jdbnet.icetray.metadata

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class MetadataFetcherTest {
    private val fetcher = MetadataFetcher()
    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun parseLegacySingleSourceObject() {
        val body = """{"icestats":{"source":{"listeners":11,"listenurl":"http://streaming.eguzki.eus:8000/eguzki.mp3","server_name":"Eguzki Irratia","genre":"denetarik"}}}"""
        val root = json.parseToJsonElement(body).jsonObject
        val source = root["icestats"]!!.jsonObject["source"]!!
        val entries = invokeParseSourceEntries(source)
        assertEquals(1, entries.size)
    }

    @Test
    fun matchSourceDisplayTitle() {
        val raw = json.parseToJsonElement(
            """
            {
              "/stream.mp3": {
                "display-title": "Current Track",
                "server_name": "My Station",
                "listeners": 5,
                "listenurl": "https://example.com/stream.mp3"
              }
            }
            """.trimIndent(),
        )
        val entries = invokeParseSourceEntries(raw)
        val np = invokeMatchSource(entries, "/stream.mp3")
        assertEquals("Current Track", np.title)
        assertEquals("My Station", np.station)
        assertEquals(5, np.listeners)
    }

    @Test
    fun parseStreamTitle() {
        val title = invokeParseStreamTitle("StreamTitle='Artist - Song';StreamUrl='http://example.com';")
        assertEquals("Artist - Song", title)
    }

    @Test
    fun publicStatsSampleParsing() {
        val body = """[{"name":"icestats","ns":"http://icecast.org/specs/legacystats-0.0.1"},{"source":{"/eguzki.mp3":{"display-title":"Live Show","genre":"denetarik","listeners":11,"listenurl":"http://streaming.eguzki.eus:8000/eguzki.mp3","server_name":"Eguzki Irratia"}}}]"""
        val docs = json.parseToJsonElement(body).jsonArray
        val source = docs[1].jsonObject["source"]!!
        val entries = invokeParseSourceEntries(source)
        val np = invokeMatchSource(entries, "/eguzki.mp3")
        assertEquals("Live Show", np.title)
        assertTrue(np.station.isNotBlank())
    }

    private fun invokeParseSourceEntries(source: kotlinx.serialization.json.JsonElement): List<*> {
        val method = MetadataFetcher::class.java.getDeclaredMethod(
            "parseSourceEntries",
            kotlinx.serialization.json.JsonElement::class.java,
        )
        method.isAccessible = true
        @Suppress("UNCHECKED_CAST")
        return method.invoke(fetcher, source) as List<*>
    }

    private fun invokeMatchSource(entries: List<*>, mount: String): uk.co.jdbnet.icetray.data.NowPlaying {
        val method = MetadataFetcher::class.java.getDeclaredMethod(
            "matchSource",
            List::class.java,
            String::class.java,
        )
        method.isAccessible = true
        return method.invoke(fetcher, entries, mount) as uk.co.jdbnet.icetray.data.NowPlaying
    }

    private fun invokeParseStreamTitle(meta: String): String {
        val method = MetadataFetcher::class.java.getDeclaredMethod("parseStreamTitle", String::class.java)
        method.isAccessible = true
        return method.invoke(fetcher, meta) as String
    }
}
