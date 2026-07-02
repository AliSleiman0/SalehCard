import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'app.dart';
import 'core/push/push_service.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await initFirebase(); // no-op until firebase_options.dart is configured
  runApp(const ProviderScope(child: SalehCardApp()));
}
