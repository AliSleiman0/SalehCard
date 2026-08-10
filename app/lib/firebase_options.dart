import 'package:firebase_core/firebase_core.dart' show FirebaseOptions;
import 'package:flutter/foundation.dart'
    show defaultTargetPlatform, TargetPlatform;

/// Firebase client configuration (Android + iOS).
///
/// Values come from the `salehcard-app` Firebase project (console:
/// https://console.firebase.google.com/project/salehcard-app). Firebase client
/// keys are identifiers, not secrets, so committing them is fine. To
/// regenerate: `flutterfire configure --project=salehcard-app`.
class DefaultFirebaseOptions {
  /// Resolves the [FirebaseOptions] for the platform this code is running on.
  static FirebaseOptions get currentPlatform {
    switch (defaultTargetPlatform) {
      case TargetPlatform.android:
        return android;
      case TargetPlatform.iOS:
        return ios;
      default:
        throw UnsupportedError(
          'DefaultFirebaseOptions are not supported for this platform.',
        );
    }
  }

  static const FirebaseOptions android = FirebaseOptions(
    apiKey: 'AIzaSyCkCvsyiMWcL9Hw3soGkdJeb8jhS2AuUsQ',
    // The `com.flashcashglobal.app` client (new package — `flashcash.global`
    // was permanently claimed by a discarded Play developer account and
    // could not be reused; that old `com.salehcard.salehcard_app` client
    // (…259dab2d…) still exists in the project for legacy installs).
    appId: '1:184899958988:android:d242dff62ae6af75742a69',
    messagingSenderId: '184899958988',
    projectId: 'salehcard-app',
    storageBucket: 'salehcard-app.firebasestorage.app',
  );

  static const FirebaseOptions ios = FirebaseOptions(
    apiKey: 'AIzaSyDAkV4b49KF6uTQXvEv6OPIKFf0saDza-g',
    appId: '1:184899958988:ios:f7a2d2e102206b2b742a69',
    messagingSenderId: '184899958988',
    projectId: 'salehcard-app',
    storageBucket: 'salehcard-app.firebasestorage.app',
    iosBundleId: 'com.flashcashglobal.app',
  );

  /// True once the placeholders have been swapped for real values.
  static bool get isConfigured => android.apiKey != 'REPLACE_ME';
}
