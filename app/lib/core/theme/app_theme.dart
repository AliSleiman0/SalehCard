import 'package:flutter/material.dart';

import 'app_colors.dart';
import 'app_tokens.dart';

/// Central theme for the SalehCard mobile app. Builds light and dark
/// [ThemeData] from [AppColors] + [AppTokens] so the whole app matches the
/// Claude Design auth system.
class AppTheme {
  const AppTheme._();

  static ThemeData light() => _build(Brightness.light, AppColors.light);
  static ThemeData dark() => _build(Brightness.dark, AppColors.dark);

  static ThemeData _build(Brightness brightness, AppColors c) {
    final colorScheme = ColorScheme.fromSeed(
      seedColor: AppTokens.brand2,
      brightness: brightness,
    ).copyWith(
      primary: AppTokens.brand2,
      secondary: AppTokens.brand1,
      tertiary: AppTokens.accent,
      surface: c.surface,
      onSurface: c.text,
      error: AppTokens.danger,
    );

    final base = ThemeData(
      useMaterial3: true,
      brightness: brightness,
      colorScheme: colorScheme,
      scaffoldBackgroundColor: c.surface,
      extensions: [c],
    );

    return base.copyWith(
      textTheme: base.textTheme.apply(
        bodyColor: c.text,
        displayColor: c.text,
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: c.surface,
        contentPadding: const EdgeInsets.symmetric(horizontal: 16),
        hintStyle: TextStyle(color: c.textFaint),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppTokens.rMd),
          borderSide: BorderSide(color: c.border),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppTokens.rMd),
          borderSide: const BorderSide(color: AppTokens.brand1, width: 1.6),
        ),
        errorBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppTokens.rMd),
          borderSide: const BorderSide(color: AppTokens.danger),
        ),
        focusedErrorBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppTokens.rMd),
          borderSide: const BorderSide(color: AppTokens.danger, width: 1.6),
        ),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: AppTokens.cta,
          foregroundColor: Colors.white,
          minimumSize: const Size.fromHeight(56),
          textStyle: const TextStyle(fontSize: 17, fontWeight: FontWeight.w800),
          shape: const StadiumBorder(),
        ),
      ),
      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(foregroundColor: AppTokens.brand1),
      ),
    );
  }
}
