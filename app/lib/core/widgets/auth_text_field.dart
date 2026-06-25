import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../theme/app_colors.dart';
import '../theme/app_tokens.dart';

/// Labeled text field for the auth screens, with the design's focus glow ring
/// (blue, or danger when in error) and an inline error message. Error state is
/// driven externally by the screen (matching the design's submit-validates
/// model), not by a Form validator.
class AuthTextField extends StatefulWidget {
  const AuthTextField({
    super.key,
    required this.label,
    required this.controller,
    this.hintText,
    this.errorText,
    this.obscureText = false,
    this.keyboardType,
    this.textInputAction,
    this.inputFormatters,
    this.autofillHints,
    this.onChanged,
    this.onSubmitted,
    this.leading,
  });

  final String label;
  final TextEditingController controller;
  final String? hintText;
  final String? errorText;
  final bool obscureText;
  final TextInputType? keyboardType;
  final TextInputAction? textInputAction;
  final List<TextInputFormatter>? inputFormatters;
  final Iterable<String>? autofillHints;
  final ValueChanged<String>? onChanged;
  final ValueChanged<String>? onSubmitted;

  /// Optional leading widget rendered inside the row before the input
  /// (e.g. a country-code box).
  final Widget? leading;

  @override
  State<AuthTextField> createState() => _AuthTextFieldState();
}

class _AuthTextFieldState extends State<AuthTextField> {
  final _focusNode = FocusNode();

  @override
  void initState() {
    super.initState();
    _focusNode.addListener(() => setState(() {}));
  }

  @override
  void dispose() {
    _focusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final hasError = widget.errorText != null;
    final focused = _focusNode.hasFocus;

    final borderColor = hasError
        ? AppTokens.danger
        : focused
            ? AppTokens.brand1
            : colors.border;

    final glow = focused
        ? [
            BoxShadow(
              color: (hasError ? AppTokens.danger : AppTokens.brand1)
                  .withValues(alpha: hasError ? 0.16 : 0.15),
              spreadRadius: 4,
              blurRadius: 0,
            ),
          ]
        : const <BoxShadow>[];

    final field = DecoratedBox(
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(AppTokens.rMd),
        border: Border.all(
          color: borderColor,
          width: focused || hasError ? 1.6 : 1,
        ),
        boxShadow: glow,
      ),
      // Height comes from equal vertical padding (≈56px total) so the text is
      // reliably centered — a fixed-height box + isDense/isCollapsed top-aligns.
      child: TextField(
        controller: widget.controller,
        focusNode: _focusNode,
        obscureText: widget.obscureText,
        keyboardType: widget.keyboardType,
        textInputAction: widget.textInputAction,
        inputFormatters: widget.inputFormatters,
        autofillHints: widget.autofillHints,
        onChanged: widget.onChanged,
        onSubmitted: widget.onSubmitted,
        style: TextStyle(fontSize: 16, height: 1.2, color: colors.text),
        cursorColor: AppTokens.brand1,
        decoration: InputDecoration(
          filled: false,
          isDense: true,
          contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 17),
          border: InputBorder.none,
          hintText: widget.hintText,
          hintStyle: TextStyle(color: colors.textFaint, height: 1.2),
        ),
      ),
    );

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          widget.label,
          style: TextStyle(
            fontSize: 14,
            fontWeight: FontWeight.w600,
            color: colors.text,
          ),
        ),
        const SizedBox(height: 9),
        if (widget.leading != null)
          Row(
            crossAxisAlignment: CrossAxisAlignment.center,
            children: [
              widget.leading!,
              const SizedBox(width: 12),
              Expanded(child: field),
            ],
          )
        else
          field,
        if (hasError) ...[
          const SizedBox(height: 7),
          Text(
            widget.errorText!,
            style: const TextStyle(
              fontSize: 13,
              fontWeight: FontWeight.w500,
              color: AppTokens.danger,
            ),
          ),
        ],
      ],
    );
  }
}
