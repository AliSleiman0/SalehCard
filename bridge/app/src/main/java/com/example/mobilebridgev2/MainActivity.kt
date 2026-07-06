package com.example.mobilebridgev2

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import com.example.mobilebridgev2.service.BridgeForegroundService
import com.example.mobilebridgev2.ui.theme.MobileBridgeV2Theme

class MainActivity : ComponentActivity() {

    private var permissionStatusMessage by mutableStateOf(
        "Telecom permissions have not been checked"
    )

    private val permissionLauncher =
        registerForActivityResult(
            ActivityResultContracts.RequestMultiplePermissions()
        ) {
            if (hasRequiredPermissions()) {
                permissionStatusMessage =
                    "Permissions granted. MobileBridge service is running."

                startBridgeService()
            } else {
                permissionStatusMessage =
                    "Required permissions were denied. MobileBridge cannot start."
            }
        }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        enableEdgeToEdge()

        setContent {
            MobileBridgeV2Theme {
                Scaffold(
                    modifier = Modifier.fillMaxSize()
                ) { innerPadding ->

                    Column(
                        modifier = Modifier
                            .fillMaxSize()
                            .padding(innerPadding)
                            .padding(24.dp),
                        verticalArrangement = Arrangement.Center,
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Text(
                            text = "MobileBridge"
                        )

                        Text(
                            text = permissionStatusMessage,
                            modifier = Modifier.padding(top = 16.dp)
                        )

                        Button(
                            onClick = {
                                checkPermissionsAndStartService()
                            },
                            modifier = Modifier.padding(top = 24.dp)
                        ) {
                            Text("Start MobileBridge")
                        }
                    }
                }
            }
        }

        checkPermissionsAndStartService()
    }

    private fun checkPermissionsAndStartService() {
        if (hasRequiredPermissions()) {
            permissionStatusMessage =
                "Permissions granted. MobileBridge service is running."

            startBridgeService()
        } else {
            permissionStatusMessage =
                "MobileBridge requires telecom permissions."

            permissionLauncher.launch(
                getRequiredPermissions()
            )
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
            ContextCompat.checkSelfPermission(
                this,
                permission
            ) == PackageManager.PERMISSION_GRANTED
        }
    }

    private fun startBridgeService() {
        val serviceIntent = Intent(
            this,
            BridgeForegroundService::class.java
        )

        ContextCompat.startForegroundService(
            this,
            serviceIntent
        )
    }
}