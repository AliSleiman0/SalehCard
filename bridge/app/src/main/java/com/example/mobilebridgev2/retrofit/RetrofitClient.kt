package com.example.mobilebridgev2.retrofit

import okhttp3.OkHttpClient
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory

object RetrofitClient {
    // TODO: Replace with HTTPS production base URL when backend TLS/auth is ready.
    const val LOCAL_HTTP_BASE_URL = "http://10.0.2.2:3000/"

    // TODO: Load a real per-device token from secure storage/provisioning.
    private const val DEVICE_TOKEN = ""

    private val httpClient: OkHttpClient by lazy {
        OkHttpClient.Builder()
            .addInterceptor { chain ->
                val original = chain.request()
                val requestBuilder = original.newBuilder()

                if (DEVICE_TOKEN.isNotBlank()) {
                    requestBuilder.header("Authorization", "Bearer $DEVICE_TOKEN")
                }

                chain.proceed(requestBuilder.build())
            }
            .build()
    }

    val api: BridgeApi by lazy {
        Retrofit.Builder()
            .baseUrl(LOCAL_HTTP_BASE_URL)
            .client(httpClient)
            .addConverterFactory(GsonConverterFactory.create())
            .build()
            .create(BridgeApi::class.java)
    }
}