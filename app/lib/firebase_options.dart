import 'package:firebase_core/firebase_core.dart';

/// Firebase client configuration (Android only — iOS is not a target).
///
/// These are PLACEHOLDER values: replace them with the real project's before
/// push notifications can work. Either run `flutterfire configure`, or copy
/// the values from the Firebase console's `google-services.json`:
///   apiKey            = client[].api_key[0].current_key
///   appId             = client[].client_info.mobilesdk_app_id
///   messagingSenderId = project_info.project_number
///   projectId         = project_info.project_id
///
/// Android Firebase client keys are identifiers, not secrets — committing the
/// real values is fine. While these placeholders are in place the app runs
/// normally with push disabled (see initFirebase in core/push/push_service.dart).
class DefaultFirebaseOptions {
  static const FirebaseOptions android = FirebaseOptions(
    apiKey: 'REPLACE_ME',
    appId: 'REPLACE_ME',
    messagingSenderId: 'REPLACE_ME',
    projectId: 'REPLACE_ME',
  );

  /// True once the placeholders have been swapped for real values.
  static bool get isConfigured => android.apiKey != 'REPLACE_ME';
}
