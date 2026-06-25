import 'package:flutter/material.dart';

import '../theme/app_tokens.dart';

/// Full-width pill CTA used across the auth screens. Shows a spinner while
/// [loading] and keeps its filled appearance (disabled but still branded).
class PrimaryCta extends StatelessWidget {
  const PrimaryCta({
    super.key,
    required this.label,
    required this.onPressed,
    this.loading = false,
  });

  final String label;
  final VoidCallback onPressed;
  final bool loading;

  @override
  Widget build(BuildContext context) {
    return FilledButton(
      onPressed: loading ? null : onPressed,
      style: FilledButton.styleFrom(
        backgroundColor: AppTokens.cta,
        disabledBackgroundColor: AppTokens.cta.withValues(alpha: 0.92),
        disabledForegroundColor: Colors.white,
        foregroundColor: Colors.white,
        elevation: 8,
        shadowColor: AppTokens.cta.withValues(alpha: 0.4),
      ),
      child: loading
          ? const SizedBox(
              height: 22,
              width: 22,
              child: CircularProgressIndicator(
                strokeWidth: 2.5,
                color: Colors.white,
              ),
            )
          : Text(label),
    );
  }
}
