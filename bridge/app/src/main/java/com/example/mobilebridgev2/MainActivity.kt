package com.example.mobilebridgev2

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import android.view.WindowManager
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import com.example.mobilebridgev2.config.DeviceConfigStore
import com.example.mobilebridgev2.service.BridgeForegroundService
import com.example.mobilebridgev2.ui.theme.MobileBridgeV2Theme
import com.example.mobilebridgev2.ussd.AccessibilityUtil
import com.example.mobilebridgev2.ussd.AlfaUssdAccessibilityService

class MainActivity : ComponentActivity() {

    private var permissionStatusMessage by mutableStateOf(
        "Telecom permissions have not been checked"
    )

    // Special-access grants the interactive Alfa-recharge USSD flow depends on. These are not
    // runtime permissions (they live in Settings), so they're surfaced here as advisory status
    // rows; the hard gate is the rechargeAlfa preflight, which fails a recharge honestly if the
    // accessibility service is off.
    private var accessibilityEnabled by mutableStateOf(false)
    private var overlayGranted by mutableStateOf(false)

    private val permissionLauncher =
        registerForActivityResult(
            ActivityResultContracts.RequestMultiplePermissions()
        ) {
            if (hasRequiredPermissions()) {
                permissionStatusMessage = "Permissions granted. Bridge service is running."
                startBridgeService()
            } else {
                permissionStatusMessage = "Required permissions were denied. Bridge cannot start."
            }
        }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        DeviceConfigStore.init(this)
        // Keep the screen on while the bridge UI is in the foreground — the dedicated bridge phone
        // sits in a drawer, and an awake screen keeps the USSD dialogs able to render.
        window.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)
        enableEdgeToEdge()

        setContent {
            MobileBridgeV2Theme {
                Scaffold(modifier = Modifier.fillMaxSize()) { innerPadding ->
                    var serverUrl by remember { mutableStateOf(DeviceConfigStore.baseUrl) }
                    var token by remember { mutableStateOf(DeviceConfigStore.token) }
                    var saved by remember { mutableStateOf("") }

                    Column(
                        modifier = Modifier
                            .fillMaxSize()
                            .padding(innerPadding)
                            .padding(24.dp)
                            .verticalScroll(rememberScrollState()),
                        verticalArrangement = Arrangement.spacedBy(14.dp),
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Text("MobileBridge")

                        OutlinedTextField(
                            value = serverUrl,
                            onValueChange = { serverUrl = it },
                            label = { Text("Server URL (https://…)") },
                            singleLine = true,
                            modifier = Modifier.fillMaxWidth()
                        )
                        OutlinedTextField(
                            value = token,
                            onValueChange = { token = it },
                            label = { Text("Device token (bd_…)") },
                            singleLine = true,
                            modifier = Modifier.fillMaxWidth()
                        )
                        Button(
                            onClick = {
                                DeviceConfigStore.baseUrl = serverUrl.trim()
                                DeviceConfigStore.token = token.trim()
                                saved = if (DeviceConfigStore.isProvisioned())
                                    "Saved. Start the bridge below." else "Enter both a URL and a token."
                            },
                            modifier = Modifier.fillMaxWidth()
                        ) { Text("Save provisioning") }

                        if (saved.isNotEmpty()) Text(saved)

                        Text(permissionStatusMessage, modifier = Modifier.padding(top = 8.dp))

                        Button(
                            onClick = { checkPermissionsAndStartService() },
                            modifier = Modifier.fillMaxWidth()
                        ) { Text("Start MobileBridge") }

                        Text(
                            if (accessibilityEnabled) "Accessibility service: ENABLED"
                            else "Accessibility service: DISABLED — required for Alfa recharge",
                            modifier = Modifier.padding(top = 8.dp)
                        )
                        Button(
                            onClick = { startActivity(AccessibilityUtil.openAccessibilitySettings()) },
                            modifier = Modifier.fillMaxWidth()
                        ) { Text("Open Accessibility settings") }

                        Text(
                            if (overlayGranted) "Display over other apps: GRANTED"
                            else "Display over other apps: DENIED — needed when screen is off"
                        )
                        Button(
                            onClick = {
                                startActivity(AccessibilityUtil.openOverlaySettings(this@MainActivity))
                            },
                            modifier = Modifier.fillMaxWidth()
                        ) { Text("Open overlay settings") }
                    }
                }
            }
        }
    }

    override fun onResume() {
        super.onResume()
        refreshSpecialAccess()
    }

    private fun refreshSpecialAccess() {
        accessibilityEnabled =
            AccessibilityUtil.isServiceEnabled(this, AlfaUssdAccessibilityService::class.java)
        overlayGranted = AccessibilityUtil.overlayGranted(this)
    }

    private fun checkPermissionsAndStartService() {
        if (!DeviceConfigStore.isProvisioned()) {
            permissionStatusMessage = "Save the server URL and device token first."
            return
        }
        if (hasRequiredPermissions()) {
            permissionStatusMessage = "Permissions granted. Bridge service is running."
            startBridgeService()
        } else {
            permissionStatusMessage = "MobileBridge requires telecom permissions."
            permissionLauncher.launch(getRequiredPermissions())
        }
    }

    private fun getRequiredPermissions(): Array<String> {
        val permissions = mutableListOf(
            Manifest.permission.CALL_PHONE,
            Manifest.permission.SEND_SMS,
            Manifest.permission.READ_PHONE_STATE,
            Manifest.permission.RECEIVE_SMS,
            Manifest.permission.READ_SMS
        )
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            permissions += Manifest.permission.POST_NOTIFICATIONS
        }
        return permissions.toTypedArray()
    }

    private fun hasRequiredPermissions(): Boolean {
        return getRequiredPermissions().all { permission ->
            ContextCompat.checkSelfPermission(this, permission) == PackageManager.PERMISSION_GRANTED
        }
    }

    private fun startBridgeService() {
        ContextCompat.startForegroundService(
            this,
            Intent(this, BridgeForegroundService::class.java)
        )
    }
}
