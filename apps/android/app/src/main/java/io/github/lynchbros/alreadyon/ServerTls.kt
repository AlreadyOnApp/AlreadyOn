package io.github.lynchbros.alreadyon

import okhttp3.HttpUrl
import okhttp3.OkHttpClient
import java.io.IOException
import java.security.SecureRandom
import java.security.cert.X509Certificate
import javax.net.ssl.SSLContext
import javax.net.ssl.X509TrustManager

/** Explicit per-server exception; never modify global TLS defaults or allow another origin. */
internal fun OkHttpClient.Builder.serverTls(origin: HttpUrl, bypass: Boolean): OkHttpClient.Builder {
    if (!bypass) return this
    val trust = object : X509TrustManager {
        override fun getAcceptedIssuers(): Array<X509Certificate> = emptyArray()
        override fun checkClientTrusted(chain: Array<X509Certificate>, authType: String) = Unit
        override fun checkServerTrusted(chain: Array<X509Certificate>, authType: String) = Unit
    }
    val context = SSLContext.getInstance("TLS").apply { init(null, arrayOf(trust), SecureRandom()) }
    return sslSocketFactory(context.socketFactory, trust)
        .hostnameVerifier { host, _ -> host == origin.host }
        .followRedirects(false).followSslRedirects(false)
        .addInterceptor { chain ->
            if (!sameOrigin(origin, chain.request().url)) throw IOException("Unexpected server origin.")
            chain.proceed(chain.request())
        }
}
