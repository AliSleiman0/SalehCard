import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../theme/app_colors.dart';
import '../theme/app_tokens.dart';

/// Six-cell OTP entry with auto-advance and backspace-to-previous. Always laid
/// out left-to-right (digits stay LTR even in an RTL locale, per the design).
class OtpInput extends StatefulWidget {
  const OtpInput({
    super.key,
    this.length = 6,
    required this.onChanged,
    this.hasError = false,
  });

  final int length;
  final ValueChanged<String> onChanged;
  final bool hasError;

  @override
  State<OtpInput> createState() => _OtpInputState();
}

class _OtpInputState extends State<OtpInput> {
  late final List<TextEditingController> _controllers;
  late final List<FocusNode> _nodes;

  @override
  void initState() {
    super.initState();
    _controllers =
        List.generate(widget.length, (_) => TextEditingController());
    _nodes = List.generate(widget.length, (_) => FocusNode());
    for (final n in _nodes) {
      n.addListener(() => setState(() {}));
    }
  }

  @override
  void dispose() {
    for (final c in _controllers) {
      c.dispose();
    }
    for (final n in _nodes) {
      n.dispose();
    }
    super.dispose();
  }

  void _emit() => widget.onChanged(_controllers.map((c) => c.text).join());

  void _onChanged(int i, String value) {
    final digits = value.replaceAll(RegExp(r'\D'), '');
    if (digits.isEmpty) {
      _controllers[i].text = '';
      _emit();
      return;
    }
    _controllers[i].text = digits.substring(digits.length - 1);
    _controllers[i].selection =
        const TextSelection.collapsed(offset: 1);
    if (i < widget.length - 1) _nodes[i + 1].requestFocus();
    _emit();
  }

  void _onKey(int i, KeyEvent event) {
    if (event is KeyDownEvent &&
        event.logicalKey == LogicalKeyboardKey.backspace &&
        _controllers[i].text.isEmpty &&
        i > 0) {
      _nodes[i - 1].requestFocus();
      _controllers[i - 1].clear();
      _emit();
    }
  }

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Directionality(
      textDirection: TextDirection.ltr,
      child: Row(
        children: [
          for (var i = 0; i < widget.length; i++) ...[
            Expanded(
              child: KeyboardListener(
                focusNode: FocusNode(skipTraversal: true),
                onKeyEvent: (e) => _onKey(i, e),
                child: _Cell(
                  controller: _controllers[i],
                  node: _nodes[i],
                  focused: _nodes[i].hasFocus,
                  hasError: widget.hasError,
                  colors: colors,
                  onChanged: (v) => _onChanged(i, v),
                ),
              ),
            ),
            if (i < widget.length - 1) const SizedBox(width: 9),
          ],
        ],
      ),
    );
  }
}

class _Cell extends StatelessWidget {
  const _Cell({
    required this.controller,
    required this.node,
    required this.focused,
    required this.hasError,
    required this.colors,
    required this.onChanged,
  });

  final TextEditingController controller;
  final FocusNode node;
  final bool focused;
  final bool hasError;
  final AppColors colors;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    final borderColor = hasError
        ? AppTokens.danger
        : focused
            ? AppTokens.brand1
            : colors.border;
    return Container(
      height: 60,
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(AppTokens.rMd),
        border: Border.all(
          color: borderColor,
          width: focused || hasError ? 1.6 : 1,
        ),
        boxShadow: focused
            ? [
                BoxShadow(
                  color: AppTokens.brand1.withValues(alpha: 0.15),
                  spreadRadius: 4,
                  blurRadius: 0,
                ),
              ]
            : null,
      ),
      alignment: Alignment.center,
      child: TextField(
        controller: controller,
        focusNode: node,
        textAlign: TextAlign.center,
        textAlignVertical: TextAlignVertical.center,
        keyboardType: TextInputType.number,
        maxLength: 1,
        onChanged: onChanged,
        style: TextStyle(
          fontSize: 24,
          fontWeight: FontWeight.w700,
          color: colors.text,
        ),
        cursorColor: AppTokens.brand1,
        decoration: const InputDecoration(
          counterText: '',
          isDense: true,
          filled: false,
          // Null out every state border so the app-wide inputDecorationTheme
          // (which defines enabledBorder/focusedBorder) can't paint a second
          // rounded outline inside the _Cell container. border: none alone does
          // not override those state borders.
          border: InputBorder.none,
          enabledBorder: InputBorder.none,
          focusedBorder: InputBorder.none,
          errorBorder: InputBorder.none,
          focusedErrorBorder: InputBorder.none,
          contentPadding: EdgeInsets.zero,
        ),
      ),
    );
  }
}
