#!/usr/bin/env python3
"""
Claude Code Statusline Simulator for Niyantra Development

Writes realistic mock statusline JSON to ~/.niyantra/data/claude-statusline.json
so you can test Niyantra's Claude Code tracking without a subscription.

Usage:
  python scripts/mock_claude_statusline.py              # Write once
  python scripts/mock_claude_statusline.py --live        # Update every 10s (simulates active session)
  python scripts/mock_claude_statusline.py --drain       # Simulate quota draining over time
  python scripts/mock_claude_statusline.py --exhausted   # Simulate near-exhausted quota
"""

import json
import os
import sys
import time
import random
from pathlib import Path

DATA_DIR = Path.home() / ".niyantra" / "data"
STATUS_FILE = DATA_DIR / "claude-statusline.json"


def make_statusline(five_hour_pct: float, seven_day_pct: float,
                    five_hour_reset_offset: int = 18000,
                    seven_day_reset_offset: int = 604800,
                    model: str = "claude-sonnet-4-6",
                    context_pct: float = 35.0,
                    cost_usd: float = 0.42) -> dict:
    """Build a realistic Claude Code statusline JSON payload."""
    now = int(time.time())
    return {
        "model": {
            "display_name": model
        },
        "workspace": {
            "git_branch": "main"
        },
        "context_window": {
            "used_percentage": context_pct
        },
        "cost": {
            "total_cost_usd": cost_usd
        },
        "rate_limits": {
            "five_hour": {
                "used_percentage": five_hour_pct,
                "resets_at": now + five_hour_reset_offset
            },
            "seven_day": {
                "used_percentage": seven_day_pct,
                "resets_at": now + seven_day_reset_offset
            }
        }
    }


def write_statusline(data: dict):
    """Write statusline JSON atomically."""
    DATA_DIR.mkdir(parents=True, exist_ok=True)
    tmp = STATUS_FILE.with_suffix(".tmp")
    with open(tmp, "w") as f:
        json.dump(data, f)
    tmp.replace(STATUS_FILE)
    print(f"[{time.strftime('%H:%M:%S')}] Wrote: 5h={data['rate_limits']['five_hour']['used_percentage']:.1f}%  "
          f"7d={data['rate_limits']['seven_day']['used_percentage']:.1f}%  "
          f"ctx={data['context_window']['used_percentage']:.1f}%  "
          f"cost=${data['cost']['total_cost_usd']:.2f}")


def main():
    mode = sys.argv[1] if len(sys.argv) > 1 else "--once"

    if mode == "--once":
        # Single snapshot: moderate usage
        data = make_statusline(42.0, 15.0, cost_usd=1.23)
        write_statusline(data)
        print(f"\nFile written to: {STATUS_FILE}")
        print("Niyantra will pick this up on next poll cycle (or click 'Snap Now').")

    elif mode == "--exhausted":
        # Near-exhausted: triggers alerts
        data = make_statusline(92.0, 78.0, cost_usd=8.50, context_pct=85.0)
        write_statusline(data)
        print(f"\nFile written to: {STATUS_FILE}")
        print("This should trigger low-quota alerts in Niyantra.")

    elif mode == "--live":
        # Simulate active session: fluctuate every 10s
        print("Simulating active Claude Code session (Ctrl+C to stop)...")
        five_pct = 20.0
        seven_pct = 8.0
        cost = 0.10
        try:
            while True:
                data = make_statusline(
                    five_pct, seven_pct,
                    cost_usd=round(cost, 2),
                    context_pct=round(random.uniform(20, 60), 1)
                )
                write_statusline(data)
                # Small random increment each cycle
                five_pct = min(100, five_pct + random.uniform(0.2, 1.5))
                seven_pct = min(100, seven_pct + random.uniform(0.05, 0.3))
                cost += random.uniform(0.01, 0.08)
                time.sleep(10)
        except KeyboardInterrupt:
            print("\nStopped.")

    elif mode == "--drain":
        # Simulate rapid quota drain (good for testing alerts)
        print("Simulating rapid quota drain (Ctrl+C to stop)...")
        five_pct = 50.0
        seven_pct = 30.0
        cost = 2.00
        try:
            while five_pct < 100:
                data = make_statusline(
                    round(five_pct, 1), round(seven_pct, 1),
                    cost_usd=round(cost, 2),
                    context_pct=round(random.uniform(40, 90), 1)
                )
                write_statusline(data)
                five_pct += random.uniform(2.0, 5.0)
                seven_pct += random.uniform(0.5, 1.5)
                cost += random.uniform(0.10, 0.30)
                time.sleep(5)
            print("\n⚠️  5-hour quota exhausted!")
        except KeyboardInterrupt:
            print("\nStopped.")

    else:
        print(f"Unknown mode: {mode}")
        print("Usage: python mock_claude_statusline.py [--once|--live|--drain|--exhausted]")
        sys.exit(1)


if __name__ == "__main__":
    main()
