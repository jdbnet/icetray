# JNI, WebView JS bridge, and media session: Go and the system look these up by name.
-keepattributes *Annotation*,InnerClasses,EnclosingMethod,Signature

-keepclasseswithmembernames class * {
    native <methods>;
}

-keepclassmembers class * {
    @android.webkit.JavascriptInterface <methods>;
}

-keep class com.wails.app.** { *; }
-keep class uk.co.jdbnet.icetray.** { *; }
