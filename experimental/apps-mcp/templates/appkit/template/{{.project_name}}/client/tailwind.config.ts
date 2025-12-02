import type { Config } from 'tailwindcss';
import tailwindcssAnimate from 'tailwindcss-animate';

const config: Config = {
  darkMode: ['class', 'media'],
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
      },
      colors: {
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        card: {
          DEFAULT: 'hsl(var(--card))',
          foreground: 'hsl(var(--card-foreground))',
        },
        popover: {
          DEFAULT: 'hsl(var(--popover))',
          foreground: 'hsl(var(--popover-foreground))',
        },
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))',
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))',
        },
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))',
        },
        accent: {
          DEFAULT: 'hsl(var(--accent))',
          foreground: 'hsl(var(--accent-foreground))',
        },
        destructive: {
          DEFAULT: 'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))',
        },
        success: {
          DEFAULT: 'hsl(var(--success))',
          foreground: 'hsl(var(--success-foreground))',
        },
        warning: {
          DEFAULT: 'hsl(var(--warning))',
          foreground: 'hsl(var(--warning-foreground))',
        },
        border: 'hsl(var(--border))',
        input: 'hsl(var(--input))',
        ring: 'hsl(var(--ring))',
        chart: {
          '1': 'hsl(var(--chart-1))',
          '2': 'hsl(var(--chart-2))',
          '3': 'hsl(var(--chart-3))',
          '4': 'hsl(var(--chart-4))',
          '5': 'hsl(var(--chart-5))',
        },
        // Databricks extended palette
        gray: {
          navigation: 'hsl(var(--gray-navigation))',
          text: 'hsl(var(--gray-text))',
          lines: 'hsl(var(--gray-lines))',
        },
        lava: {
          800: 'hsl(var(--lava-800))',
          700: 'hsl(var(--lava-700))',
          600: 'hsl(var(--lava-600))',
          500: 'hsl(var(--lava-500))',
          400: 'hsl(var(--lava-400))',
          300: 'hsl(var(--lava-300))',
        },
        navy: {
          900: 'hsl(var(--navy-900))',
          800: 'hsl(var(--navy-800))',
          700: 'hsl(var(--navy-700))',
          600: 'hsl(var(--navy-600))',
          500: 'hsl(var(--navy-500))',
          400: 'hsl(var(--navy-400))',
          300: 'hsl(var(--navy-300))',
        },
        green: {
          600: 'hsl(var(--green-600))',
          500: 'hsl(var(--green-500))',
          400: 'hsl(var(--green-400))',
        },
        blue: {
          600: 'hsl(var(--blue-600))',
          500: 'hsl(var(--blue-500))',
          400: 'hsl(var(--blue-400))',
        },
        yellow: {
          600: 'hsl(var(--yellow-600))',
          500: 'hsl(var(--yellow-500))',
          400: 'hsl(var(--yellow-400))',
        },
      },
    },
  },
  plugins: [tailwindcssAnimate],
};

export default config;
