package com.example.mobilebridgev2.retrofit

import com.example.mobilebridgev2.config.DeviceConfigStore
import okhttp3.OkHttpClient
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory

/**
 * Builds the [BridgeApi] from the device's stored provisioning (server URL +
 * token). The Retrofit instance is rebuilt whenever the base URL changes so
 * re-provisioning to a new server takes effect without restarting the app; the
 * bearer token is read per-request by the interceptor, so a token rotation is
 * picked up immediately.
 */
object RetrofitClient {
    // Emulator-loopback fallback used only before the device is provisioned.
    private const val DEV_FALLBACK = "http://10.0.2.2:3000"
    private const val PATH = "api/v1/bridge/"

    @Volatile private var cachedBase: String? = null
    @Volatile private var cachedApi: BridgeApi? = null

    private val httpClient: OkHttpClient by lazy {
        OkHttpClient.Builder()
            .addInterceptor { chain ->
                val builder = chain.request().newBuilder()
                val token = DeviceConfigStore.token
                if (token.isNotBlank()) {
                    builder.header("Authorization", "Bearer $token")
                }
                chain.proceed(builder.build())
            }
            .build()
    }

    /** Server base (scheme+host[+port]) with the bridge path appended. */
    private fun resolveBase(): String {
        var root = DeviceConfigStore.baseUrl.trim()
        if (root.isEmpty()) root = DEV_FALLBACK
        root = root.trimEnd('/')
        // Accept either a bare server root or one that already includes the path.
        return if (root.endsWith("/api/v1/bridge")) "$root/" else "$root/$PATH"
    }

    val api: BridgeApi
        @Synchronized get() {
            val base = resolveBase()
            val existing = cachedApi
            if (existing != null && base == cachedBase) return existing
            val built = Retrofit.Builder()
                .baseUrl(base)
                .client(httpClient)
                .addConverterFactory(GsonConverterFactory.create())
                .build()
                .create(BridgeApi::class.java)
            cachedBase = base
            cachedApi = built
            return built
        }
}
