import 'package:flutter/material.dart';

// ── Palette ──
const primaryColor = Color(0xFF4F6EF7);
const primaryLight = Color(0xFF818CF8);

const successColor = Color(0xFF22C55E);
const warningColor = Color(0xFFF59E0B);
const dangerColor = Color(0xFFEF4444);

const accentOrange = Color(0xFFE67E22);
const accentTeal = Color(0xFF14B8A6);
const accentPurple = Color(0xFF8B5CF6);

const bgColor = Color(0xFFF1F5F9);
const surfaceColor = Colors.white;
const surfaceSecondary = Color(0xFFF8FAFC);

// ── Text ──
const textPrimary = Color(0xFF1E293B);
const textSecondary = Color(0xFF64748B);
const textMuted = Color(0xFF94A3B8);

// ── Border ──
const borderColor = Color(0xFFE2E8F0);

// ── Radii ──
const double radiusSm = 6;
const double radiusMd = 10;
const double radiusLg = 14;

// ── Gradients ──
const Gradient primaryGradient = LinearGradient(
  colors: [primaryColor, primaryLight],
  begin: Alignment.topLeft,
  end: Alignment.bottomRight,
);

ThemeData appTheme() => ThemeData(
      useMaterial3: true,
      colorSchemeSeed: primaryColor,
      brightness: Brightness.light,
      scaffoldBackgroundColor: bgColor,
      fontFamily: 'NotoSansSC',
      appBarTheme: const AppBarTheme(
        backgroundColor: Colors.white,
        foregroundColor: textPrimary,
        elevation: 0,
        surfaceTintColor: Colors.transparent,
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: surfaceColor,
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 14, vertical: 14),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(radiusMd),
          borderSide: const BorderSide(color: borderColor),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(radiusMd),
          borderSide: const BorderSide(color: borderColor),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(radiusMd),
          borderSide: const BorderSide(color: primaryColor, width: 1.5),
        ),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: primaryColor,
          foregroundColor: Colors.white,
          shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(radiusMd)),
          padding:
              const EdgeInsets.symmetric(vertical: 14, horizontal: 24),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(radiusMd)),
          padding:
              const EdgeInsets.symmetric(vertical: 14, horizontal: 24),
        ),
      ),
    );
