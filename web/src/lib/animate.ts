import autoAnimate, { type AutoAnimateOptions } from "@formkit/auto-animate";

export function animate(node: HTMLElement, options: Partial<AutoAnimateOptions> = {}) {
  const controller = autoAnimate(node, options);
  return {
    update(next: Partial<AutoAnimateOptions>) {
      controller.update?.(next);
    },
    destroy() {
      controller.destroy?.();
    },
  };
}
