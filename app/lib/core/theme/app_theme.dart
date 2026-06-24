import 'package:flutter/material.dart';

/// Neutral placeholder theme. Real design tokens land after the user provides
/// screenshots; this is intentionally minimal Material 3.
class AppTheme {
  const AppTheme._();

  static ThemeData light() => ThemeData(
        useMaterial3: true,
        colorSchemeSeed: const Color(0xFF2563EB),
        brightness: Brightness.light,
      );
}
