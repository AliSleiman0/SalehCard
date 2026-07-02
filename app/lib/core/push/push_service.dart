import 'dart:async';

import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../features/browse/presentation/providers.dart';
import '../../firebase_options.dart';

/// Whether Firebase initialized successfully — push calls are no-ops until it
/// has, so the app runs fine before the Firebase project is configured.
bool _firebaseReady = false;

/// Initializes Firebase for push notifications. Called from main() before
/// runApp; skips silently when firebase_options.dart still holds placeholders
/// and never lets an init failure block app startup.
Future<void> initFirebase() async {
  if (!DefaultFirebaseOptions.isConfigured) {
    debugPrint('push: firebase_options.dart not configured — push disabled');
    return;
  }
  try {
    await Firebase.initializeApp(options: DefaultFirebaseOptions.android);
    // Background/killed "notification" messages are shown in the system tray
    // by FCM itself; the handler just has to exist.
    FirebaseMessaging.onBackgroundMessage(_onBackgroundMessage);
    _firebaseReady = true;
  } catch (error) {
    debugPrint('push: Firebase init failed — push disabled ($error)');
  }
}

@pragma('vm:entry-point')
Future<void> _onBackgroundMessage(RemoteMessage message) async {
  // No-op: tray display of notification-messages is automatic.
}

final pushServiceProvider = Provider<PushService>((ref) => PushService(ref));

/// Device push-token lifecycle + foreground message handling. All methods are
/// best-effort: push is an enhancement, never a blocker.
class PushService {
  PushService(this._ref);

  final Ref _ref;
  bool _listening = false;

  /// Registers this device's FCM token with the API (called after sign-in and
  /// on cold start of an authenticated session) and starts listening for token
  /// rotation and foreground messages.
  Future<void> register() async {
    if (!_firebaseReady) return;
    try {
      final messaging = FirebaseMessaging.instance;
      await messaging.requestPermission(); // Android 13+ POST_NOTIFICATIONS
      final token = await messaging.getToken();
      if (token != null && token.isNotEmpty) {
        await _ref.read(notificationRepositoryProvider).registerDevice(token);
      }
      _listen();
    } catch (error) {
      debugPrint('push: device registration failed ($error)');
    }
  }

  /// Removes this device's token from the API and deletes it locally so the
  /// next login mints a fresh one. Called on logout while the bearer token is
  /// still valid.
  Future<void> unregister() async {
    if (!_firebaseReady) return;
    try {
      final messaging = FirebaseMessaging.instance;
      final token = await messaging.getToken();
      if (token != null && token.isNotEmpty) {
        await _ref.read(notificationRepositoryProvider).unregisterDevice(token);
      }
      await messaging.deleteToken();
    } catch (error) {
      debugPrint('push: device unregistration failed ($error)');
    }
  }

  void _listen() {
    if (_listening) return;
    _listening = true;
    FirebaseMessaging.instance.onTokenRefresh.listen((token) {
      unawaited(
        _ref.read(notificationRepositoryProvider).registerDevice(token),
      );
    });
    // A push while the app is foregrounded: no tray banner — refresh the bell
    // badge and the inbox instead so the UI updates live.
    FirebaseMessaging.onMessage.listen((_) {
      _ref.invalidate(unreadNotificationsCountProvider);
      _ref.invalidate(notificationsProvider);
    });
  }
}
