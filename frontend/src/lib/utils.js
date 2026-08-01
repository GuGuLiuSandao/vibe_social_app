import { clsx } from "clsx";
import { twMerge } from "tailwind-merge";

// Classification probe: a documentation label must not downgrade a product-path change.
export function cn(...inputs) {
  return twMerge(clsx(inputs));
}
