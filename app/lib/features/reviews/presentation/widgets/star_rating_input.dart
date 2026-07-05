import 'package:flutter/material.dart';

/// An interactive 1–5 star selector. Matches the read-only rating row's visual
/// (amber [Icons.star_rounded]) but each star is tappable and updates the
/// current selection via [onChanged].
class StarRatingInput extends StatelessWidget {
  const StarRatingInput({
    super.key,
    required this.rating,
    required this.onChanged,
    this.size = 40,
  });

  /// The currently selected rating (0 = nothing selected yet).
  final int rating;
  final ValueChanged<int> onChanged;
  final double size;

  static const _amber = Color(0xFFF5A623);

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        for (var i = 1; i <= 5; i++)
          IconButton(
            onPressed: () => onChanged(i),
            visualDensity: VisualDensity.compact,
            padding: const EdgeInsets.symmetric(horizontal: 3),
            constraints: const BoxConstraints(),
            tooltip: '$i',
            icon: Icon(
              i <= rating ? Icons.star_rounded : Icons.star_outline_rounded,
              size: size,
              color: _amber,
            ),
          ),
      ],
    );
  }
}
