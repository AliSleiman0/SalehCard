import 'dart:async';

import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../features/browse/presentation/providers.dart';
import '../../features/complaints/data/complaint_repository.dart';
import '../../features/complaints/domain/complaint.dart';
import '../router/app_router.dart';
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
    await Firebase.initializeApp(
      options: DefaultFirebaseOptions.currentPlatform,
    );
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
    // badge and the inbox instead so the UI updates live (and an open
    // complaint thread / list, for complaint pushes).
    FirebaseMessaging.onMessage.listen((message) {
      _ref.invalidate(unreadNotificationsCountProvider);
      _ref.invalidate(notificationsProvider);
      if (complaintNotificationKinds.contains(message.data['kind'])) {
        _ref.invalidate(complaintsProvider);
        _ref.invalidate(complaintDetailProvider);
      }
    });
    // Tapping a tray notification while the app was in the background.
    FirebaseMessaging.onMessageOpenedApp.listen(
      (message) => _open(message, coldStart: false),
    );
    // Tapping a tray notification that launched the app from killed. register()
    // runs once the session is authenticated, so the router guard lets it through.
    unawaited(
      FirebaseMessaging.instance.getInitialMessage().then((message) {
        if (message != null) _open(message, coldStart: true);
      }),
    );
  }

  /// Routes a tapped notification to its screen. Only complaint notifications
  /// deep-link today; every other kind just opens the app as before.
  void _open(RemoteMessage message, {required bool coldStart}) {
    final route = pushRouteFor(message.data);
    if (route == null) return;
    _ref.invalidate(complaintDetailProvider);
    final router = _ref.read(routerProvider);
    if (coldStart) {
      // Land on home first so back from the complaint has somewhere to go.
      router.go('/home');
    }
    router.push(route);
  }
}

/// The in-app route for a push payload, or null when it should just open the
/// app. The server puts the notification kind into the FCM data map next to the
/// structured values (complaintId, number, ...).
String? pushRouteFor(Map<String, dynamic> data) =>
    complaintRouteFor(data['kind']?.toString(), data);
