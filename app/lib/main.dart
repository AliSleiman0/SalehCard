import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'app.dart';
import 'core/push/push_service.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await initFirebase(); // no-op until firebase_options.dart is configured
  // Help Center bot avatar: Microsoft Fluent Emoji (MIT) — shown on the
  // system licenses page alongside the package licenses.
  LicenseRegistry.addLicense(() async* {
    final text = await rootBundle.loadString(
      'assets/help/LICENSE-fluentui-emoji.txt',
    );
    yield LicenseEntryWithLineBreaks(['fluentui-emoji'], text);
  });
  runApp(const ProviderScope(child: SalehCardApp()));
}
