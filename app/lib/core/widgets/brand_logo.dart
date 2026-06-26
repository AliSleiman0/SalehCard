import 'package:flutter/material.dart';

/// The SalehCard logo — the gamepad + lightning brand mark, optionally the full
/// "SALEH CARD" lockup. Backed by the shipped PNG assets in `assets/branding/`
/// so the launcher icon, splash, and in-app brand all share one mark.
class BrandLogo extends StatelessWidget {
  const BrandLogo({super.key, this.size = 44, this.showWordmark = false});

  /// Height of the logo in logical pixels. For the square mark this is also the
  /// width; for the lockup the width is intrinsic (the mark + wordmark are wider
  /// than tall).
  final double size;

  /// When true, renders the full mark + "SALEH CARD" lockup instead of the mark.
  final bool showWordmark;

  @override
  Widget build(BuildContext context) {
    if (showWordmark) {
      return Image.asset(
        'assets/branding/logo_lockup.png',
        height: size,
        fit: BoxFit.contain,
      );
    }
    return Image.asset(
      'assets/branding/logo_mark.png',
      height: size,
      width: size,
      fit: BoxFit.contain,
    );
  }
}
