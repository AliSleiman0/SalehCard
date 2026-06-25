import 'package:flutter/material.dart';

/// Theme-independent brand tokens for the SalehCard app (the blue→magenta
/// gradient, CTA fill, accent, danger, radii, spacing). Surface/text tokens that
/// differ between light and dark live in [AppColors].
class AppTokens {
  const AppTokens._();

  // ---- Brand ----
  static const Color brand1 = Color(0xFF3B5BFF); // electric blue
  static const Color brandMid = Color(0xFF8A3BFF); // violet (gradient midpoint)
  static const Color brand2 = Color(0xFFD633FF); // magenta
  static const Color cta = Color(0xFFA21CAF); // deep magenta — primary button
  static const Color accent = Color(0xFF22E3C8); // cyan
  static const Color danger = Color(0xFFFF4D6D);

  /// Signature brand gradient (135°), used on the logo mark and gradient text.
  static const Gradient brandGradient = LinearGradient(
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
    colors: [brand1, brandMid, brand2],
    stops: [0.0, 0.52, 1.0],
  );

  // ---- Radii ----
  static const double rSm = 10;
  static const double rMd = 16;
  static const double rLg = 22;
  static const double rPill = 999;

  // ---- Spacing ----
  static const double gap = 16;
}
