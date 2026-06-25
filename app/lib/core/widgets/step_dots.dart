import 'package:flutter/material.dart';

import '../theme/app_colors.dart';
import '../theme/app_tokens.dart';

/// Progress indicator for the multi-step sign-up flow: the active step is a wide
/// gradient pill, the rest are small dots.
class StepDots extends StatelessWidget {
  const StepDots({super.key, required this.count, required this.activeIndex});

  final int count;
  final int activeIndex;

  @override
  Widget build(BuildContext context) {
    final inactive = context.colors.borderStrong;
    return Row(
      children: [
        for (var i = 0; i < count; i++)
          Padding(
            padding: const EdgeInsetsDirectional.only(end: 6),
            child: i == activeIndex
                ? Container(
                    width: 22,
                    height: 6,
                    decoration: BoxDecoration(
                      gradient: AppTokens.brandGradient,
                      borderRadius: BorderRadius.circular(3),
                    ),
                  )
                : Container(
                    width: 6,
                    height: 6,
                    decoration: BoxDecoration(
                      color: inactive,
                      borderRadius: BorderRadius.circular(3),
                    ),
                  ),
          ),
      ],
    );
  }
}
