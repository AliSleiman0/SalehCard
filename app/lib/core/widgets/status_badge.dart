import 'package:flutter/material.dart';

import '../theme/app_tokens.dart';

/// A small colored pill for statuses (order status, KYC status, etc). The colour
/// is tinted at low opacity for the fill with a solid text/dot.
class StatusBadge extends StatelessWidget {
  const StatusBadge({super.key, required this.label, required this.color});

  final String label;
  final Color color;

  /// Convenience colours for the common semantic states.
  static const Color success = AppTokens.accent;
  static const Color warning = Color(0xFFFFB02E);
  static const Color danger = AppTokens.danger;
  static const Color neutral = Color(0xFF8C8CA6);

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.16),
        borderRadius: BorderRadius.circular(AppTokens.rPill),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 6,
            height: 6,
            decoration: BoxDecoration(color: color, shape: BoxShape.circle),
          ),
          const SizedBox(width: 6),
          Text(
            label,
            style: TextStyle(
                color: color, fontSize: 12, fontWeight: FontWeight.w700),
          ),
        ],
      ),
    );
  }
}
