package uk.co.jdbnet.icetray.images

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.net.Uri
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File
import java.io.FileOutputStream
import kotlin.math.max

class ImageStore(private val imagesDir: File) {
    suspend fun saveStreamImage(streamId: String, uri: Uri, context: Context): String = withContext(Dispatchers.IO) {
        imagesDir.mkdirs()
        context.contentResolver.openInputStream(uri)?.use { input ->
            val original = BitmapFactory.decodeStream(input)
                ?: throw IllegalArgumentException("invalid image")
            if (max(original.width, original.height) < 32) {
                throw IllegalArgumentException("image too small")
            }
            val resized = Bitmap.createScaledBitmap(original, TARGET_SIZE, TARGET_SIZE, true)
            if (resized != original) {
                original.recycle()
            }
            val filename = "$streamId.png"
            val outFile = File(imagesDir, filename)
            FileOutputStream(outFile).use { output ->
                if (!resized.compress(Bitmap.CompressFormat.PNG, 100, output)) {
                    throw IllegalStateException("failed to save image")
                }
            }
            resized.recycle()
            filename
        } ?: throw IllegalArgumentException("could not read image")
    }

    fun deleteImage(filename: String) {
        if (filename.isBlank()) return
        File(imagesDir, filename).delete()
    }

    companion object {
        private const val TARGET_SIZE = 512
    }
}
