import 'package:flutter/material.dart';

import '../theme/app_colors.dart';
import '../theme/app_tokens.dart';

/// Static Lebanon country-code box (+961) shown before the phone input. The flag
/// is drawn (not an emoji) so it renders identically across devices.
class CountryCodeBox extends StatelessWidget {
  const CountryCodeBox({super.key});

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Container(
      height: 56,
      padding: const EdgeInsets.symmetric(horizontal: 14),
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(AppTokens.rMd),
        border: Border.all(color: colors.border),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          _LebanonFlag(border: colors.border),
          const SizedBox(width: 8),
          Text(
            '+961',
            style: TextStyle(
              fontSize: 15,
              fontWeight: FontWeight.w700,
              color: colors.text,
            ),
          ),
        ],
      ),
    );
  }
}

class _LebanonFlag extends StatelessWidget {
  const _LebanonFlag({required this.border});

  final Color border;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 24,
      height: 16,
      clipBehavior: Clip.antiAlias,
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(3),
        border: Border.all(color: border),
      ),
      child: Column(
        children: [
          Expanded(flex: 1, child: Container(color: const Color(0xFFD7222E))),
          Expanded(
            flex: 2,
            child: Container(
              color: Colors.white,
              alignment: Alignment.center,
              child: Transform.rotate(
                angle: 0.785398, // 45°
                child: Container(
                  width: 5,
                  height: 5,
                  color: const Color(0xFF2E7D32),
                ),
              ),
            ),
          ),
          Expanded(flex: 1, child: Container(color: const Color(0xFFD7222E))),
        ],
      ),
    );
  }
}
