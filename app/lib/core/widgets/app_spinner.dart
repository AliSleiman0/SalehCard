import 'package:flutter/material.dart';

import '../theme/app_tokens.dart';

/// The app's single branded progress spinner. Defaults to the cyan [AppTokens.accent]
/// tint, which reads well on both the light and dark surfaces. Pass [color] for
/// on-brand-fill buttons (e.g. white on the CTA).
class AppSpinner extends StatelessWidget {
  const AppSpinner({super.key, this.size = 28, this.stroke = 3, this.color});

  /// Side length of the spinner box in logical pixels.
  final double size;

  /// Stroke width of the indicator arc.
  final double stroke;

  /// Override tint. Defaults to [AppTokens.accent].
  final Color? color;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: size,
      height: size,
      child: CircularProgressIndicator(
        strokeWidth: stroke,
        color: color ?? AppTokens.accent,
      ),
    );
  }
}

/// Full-screen centered [AppSpinner] for `AsyncValue.when(loading:)` branches.
class LoadingView extends StatelessWidget {
  const LoadingView({super.key});

  @override
  Widget build(BuildContext context) => const Center(child: AppSpinner());
}
