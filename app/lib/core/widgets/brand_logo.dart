import 'package:flutter/material.dart';

import '../theme/app_colors.dart';
import '../theme/app_tokens.dart';

/// The SalehCard logo — a gradient rounded-square "S spark" mark, optionally
/// followed by the "SalehCard" wordmark. Ported from `web/src/components/Logo.tsx`
/// so mobile and web share one brand mark.
class BrandLogo extends StatelessWidget {
  const BrandLogo({super.key, this.size = 44, this.showWordmark = false});

  /// Side length of the square mark in logical pixels.
  final double size;

  /// When true, renders "SalehCard" next to the mark.
  final bool showWordmark;

  @override
  Widget build(BuildContext context) {
    final mark = Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        gradient: AppTokens.brandGradient,
        borderRadius: BorderRadius.circular(size * 0.3),
        boxShadow: [
          BoxShadow(
            color: AppTokens.brandMid.withValues(alpha: 0.5),
            blurRadius: 18,
            offset: const Offset(0, 6),
          ),
        ],
      ),
      child: Center(
        child: CustomPaint(
          size: Size.square(size * 0.6),
          painter: _SparkMarkPainter(),
        ),
      ),
    );

    if (!showWordmark) return mark;

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        mark,
        SizedBox(width: size * 0.34),
        _Wordmark(fontSize: size * 0.5),
      ],
    );
  }
}

class _Wordmark extends StatelessWidget {
  const _Wordmark({required this.fontSize});

  final double fontSize;

  @override
  Widget build(BuildContext context) {
    final style = TextStyle(
      fontSize: fontSize,
      fontWeight: FontWeight.w800,
      letterSpacing: -0.5,
      height: 1,
      color: context.colors.text,
    );
    return Text.rich(
      TextSpan(
        children: [
          TextSpan(text: 'Saleh', style: style),
          // Gradient "Card" via a shader applied to the foreground.
          TextSpan(
            text: 'Card',
            style: style.copyWith(
              foreground: Paint()
                ..shader = AppTokens.brandGradient.createShader(
                  Rect.fromLTWH(0, 0, fontSize * 2.4, fontSize),
                ),
            ),
          ),
        ],
      ),
    );
  }
}

/// Paints the geometric "S" stroke + cyan spark dot in a 24×24 coordinate space,
/// matching the SVG path in the web logo.
class _SparkMarkPainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final scale = size.width / 24.0;
    canvas.scale(scale);

    final stroke = Paint()
      ..color = Colors.white
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2.6
      ..strokeCap = StrokeCap.round
      ..strokeJoin = StrokeJoin.round;

    // path: M16 5 H10 a3 3 0 0 0 0 6 h4 a3 3 0 0 1 0 6 H7
    final path = Path()
      ..moveTo(16, 5)
      ..lineTo(10, 5)
      ..arcToPoint(const Offset(10, 11),
          radius: const Radius.circular(3), clockwise: false)
      ..lineTo(14, 11)
      ..arcToPoint(const Offset(14, 17),
          radius: const Radius.circular(3), clockwise: true)
      ..lineTo(7, 17);
    canvas.drawPath(path, stroke);

    final dot = Paint()..color = AppTokens.accent;
    canvas.drawCircle(const Offset(18.5, 6), 1.7, dot);
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}
