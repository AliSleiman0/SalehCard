package com.example.mobilebridgev2.config

import android.content.Context

/**
 * Persists the device provisioning: the server base URL and the per-device
 * bearer token issued from the admin Bridge page. Backed by SharedPreferences.
 *
 * NOTE (v1): plain SharedPreferences. The token is a low-value, instantly
 * revocable/rotatable recharge-device credential; hardening to
 * EncryptedSharedPreferences is a deferred follow-up (see BRIDGE-PLAN.md).
 */
object DeviceConfigStore {
    private const val PREFS = "bridge_prefs"
    private const val KEY_BASE_URL = "base_url"
    private const val KEY_TOKEN = "device_token"

    @Volatile
    private var prefs: android.content.SharedPreferences? = null

    fun init(context: Context) {
        if (prefs == null) {
            prefs = context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        }
    }

    var baseUrl: String
        get() = prefs?.getString(KEY_BASE_URL, "").orEmpty()
        set(value) {
            prefs?.edit()?.putString(KEY_BASE_URL, value.trim())?.apply()
        }

    var token: String
        get() = prefs?.getString(KEY_TOKEN, "").orEmpty()
        set(value) {
            prefs?.edit()?.putString(KEY_TOKEN, value.trim())?.apply()
        }

    fun isProvisioned(): Boolean = baseUrl.isNotBlank() && token.isNotBlank()
}
