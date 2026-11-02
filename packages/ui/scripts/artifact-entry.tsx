// The kit components mirrored into the Module/01 design-system artifact. build-artifact.ts bundles
// this as a classic script that extends window.Module01 next to the portfolio components.

import { Amount } from "../src/components/amount";
import { BudgetMeter } from "../src/components/budget-meter";
import { Calendar } from "../src/components/calendar";
import { CategoryIcon, ColorDot } from "../src/components/category-icon";
import { BalanceChart, DonutChart, FlowChart } from "../src/components/charts";
import { Combobox } from "../src/components/combobox";
import { DatePicker } from "../src/components/date-picker";
import { Field, FieldError } from "../src/components/field";
import { FxConverter } from "../src/components/fx-converter";
import { Input } from "../src/components/input";
import { MoneyInput } from "../src/components/money-input";
import { MultiCombobox } from "../src/components/multi-combobox";
import { categoryColors, categoryIcons } from "../src/lib/category-icons";
import { fault } from "../src/lib/fault";
import { editFx, fxResult, initialFxState } from "../src/lib/fx";
import { formatMoney, parseAmount } from "../src/lib/money";

export const components = {
  CategoryIcon,
  ColorDot,
  Amount,
  MoneyInput,
  FxConverter,
  Field,
  FieldError,
  TextInput: Input,
  Calendar,
  DatePicker,
  Combobox,
  MultiCombobox,
  BudgetMeter,
  BalanceChart,
  FlowChart,
  DonutChart,
};

const api = {
  ...components,
  fault,
  fx: { initialFxState, editFx, fxResult },
  money: { formatMoney, parseAmount },
  categoryColors,
  categoryIconNames: Object.keys(categoryIcons),
};

const w = window as unknown as { Module01?: Record<string, unknown> };
w.Module01 = Object.assign(w.Module01 ?? {}, api);
