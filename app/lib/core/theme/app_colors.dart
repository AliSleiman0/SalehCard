import 'package:flutter/material.dart';

/// Theme-dependent surface/text/border tokens, carried as a [ThemeExtension] so
/// they switch automatically between light and dark. Brand colors (gradient,
/// CTA, accent, danger) are theme-independent and live in [AppTokens].
///
/// Access via `context.colors` (see [AppColorsX]).
@immutable
class AppColors extends ThemeExtension<AppColors> {
  const AppColors({
    required this.bg,
    required this.surface,
    required this.topbar,
    required this.text,
    required this.textDim,
    required this.textFaint,
    required this.border,
    required this.borderStrong,
  });

  final Color bg;
  final Color surface;
  final Color topbar;
  final Color text;
  final Color textDim;
  final Color textFaint;
  final Color border;
  final Color borderStrong;

  static const AppColors light = AppColors(
    bg: Color(0xFFEEF0F8),
    surface: Color(0xFFFFFFFF),
    topbar: Color(0xFFEAEAEE),
    text: Color(0xFF14141F),
    textDim: Color(0xFF565672),
    textFaint: Color(0xFF8C8CA6),
    border: Color(0x170E0E26), // rgba(14,14,38,0.09)
    borderStrong: Color(0x2914142E), // rgba(14,14,38,0.16)
  );

  static const AppColors dark = AppColors(
    bg: Color(0xFF0D0D17),
    surface: Color(0xFF16161F),
    topbar: Color(0xFF1E1E2A),
    text: Color(0xFFF4F4FA),
    textDim: Color(0xFFA6A6C2),
    textFaint: Color(0xFF6E6E8C),
    border: Color(0x1AFFFFFF), // rgba(255,255,255,0.10)
    borderStrong: Color(0x33FFFFFF), // rgba(255,255,255,0.20)
  );

  @override
  AppColors copyWith({
    Color? bg,
    Color? surface,
    Color? topbar,
    Color? text,
    Color? textDim,
    Color? textFaint,
    Color? border,
    Color? borderStrong,
  }) {
    return AppColors(
      bg: bg ?? this.bg,
      surface: surface ?? this.surface,
      topbar: topbar ?? this.topbar,
      text: text ?? this.text,
      textDim: textDim ?? this.textDim,
      textFaint: textFaint ?? this.textFaint,
      border: border ?? this.border,
      borderStrong: borderStrong ?? this.borderStrong,
    );
  }

  @override
  AppColors lerp(ThemeExtension<AppColors>? other, double t) {
    if (other is! AppColors) return this;
    return AppColors(
      bg: Color.lerp(bg, other.bg, t)!,
      surface: Color.lerp(surface, other.surface, t)!,
      topbar: Color.lerp(topbar, other.topbar, t)!,
      text: Color.lerp(text, other.text, t)!,
      textDim: Color.lerp(textDim, other.textDim, t)!,
      textFaint: Color.lerp(textFaint, other.textFaint, t)!,
      border: Color.lerp(border, other.border, t)!,
      borderStrong: Color.lerp(borderStrong, other.borderStrong, t)!,
    );
  }
}

/// Convenience accessor: `context.colors.surface`.
extension AppColorsX on BuildContext {
  AppColors get colors => Theme.of(this).extension<AppColors>()!;
}
