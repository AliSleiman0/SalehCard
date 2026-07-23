import 'package:firebase_core/firebase_core.dart';

/// Firebase client configuration (Android only — iOS is not a target).
///
/// Values come from the `salehcard-app` Firebase project (console:
/// https://console.firebase.google.com/project/salehcard-app). Android Firebase
/// client keys are identifiers, not secrets, so committing them is fine. To
/// regenerate: `firebase apps:sdkconfig ANDROID --project salehcard-app`.
class DefaultFirebaseOptions {
  static const FirebaseOptions android = FirebaseOptions(
    apiKey: 'AIzaSyCkCvsyiMWcL9Hw3soGkdJeb8jhS2AuUsQ',
    // The `flashcash.global` client. The old `com.salehcard.salehcard_app`
    // client (…259dab2d…) still exists in the project for legacy installs.
    appId: '1:184899958988:android:1b1749afb3bcb962742a69',
    messagingSenderId: '184899958988',
    projectId: 'salehcard-app',
    storageBucket: 'salehcard-app.firebasestorage.app',
  );

  /// True once the placeholders have been swapped for real values.
  static bool get isConfigured => android.apiKey != 'REPLACE_ME';
}
