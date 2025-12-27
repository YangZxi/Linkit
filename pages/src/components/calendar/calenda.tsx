import { useState } from "react";
import { DayPicker } from "react-day-picker";
import "react-day-picker/style.css";

export default function XCalendar() {
  const [selected, setSelected] = useState<Date | undefined>(undefined);

  return <div>
      <DayPicker
    animate
    mode="single"
      selected={selected}
      onSelect={setSelected}
      footer={
        selected ? `Selected: ${selected.toLocaleDateString()}` : "Pick a day."
      }
    />
    </div>
}