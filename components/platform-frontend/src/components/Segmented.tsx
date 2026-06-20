import { useState } from "react";

// Segmented toggle (.segmented) — purely visual selection, matching the prototype.
export function Segmented({
  options,
  defaultValue,
  onChange,
}: {
  options: string[];
  defaultValue?: string;
  onChange?: (v: string) => void;
}) {
  const [val, setVal] = useState(defaultValue ?? options[0]);
  return (
    <div className="segmented">
      {options.map((o) => (
        <button
          key={o}
          className={o === val ? "on" : ""}
          onClick={() => {
            setVal(o);
            onChange?.(o);
          }}
        >
          {o}
        </button>
      ))}
    </div>
  );
}
