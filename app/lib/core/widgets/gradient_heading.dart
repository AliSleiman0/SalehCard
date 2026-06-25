import 'package:flutter/material.dart';

import '../theme/app_tokens.dart';

/// Page heading painted with the brand gradient (used on the auth screens).
class GradientHeading extends StatelessWidget {
  const GradientHeading(this.text, {super.key});

  final String text;

  @override
  Widget build(BuildContext context) {
    return ShaderMask(
      shaderCallback: (bounds) => AppTokens.brandGradient.createShader(bounds),
      blendMode: BlendMode.srcIn,
      child: Text(
        text,
        style: const TextStyle(
          fontSize: 30,
          fontWeight: FontWeight.w800,
          height: 1.12,
          color: Colors.white,
        ),
      ),
    );
  }
}
